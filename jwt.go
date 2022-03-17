// SPDX-FileCopyrightText: 2022 Marshall Wace <opensource@mwam.com>
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"crypto/rsa"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
)

var (
	jwtRsaPubKey    *rsa.PublicKey
	jwtIssuerFlag   string
	jwtAudienceFlag string
)

type JwtClaims struct {
	Action string `json:"action"`
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`
	jwt.StandardClaims
}

type AuthHeader struct {
	Authorization string `header:"Authorization"`
}

func getRequestParam(c *gin.Context, paramKey string) string {
	if c.Param(paramKey) != "" {
		return c.Param(paramKey)
	} else if c.Query(paramKey) != "" {
		return c.Query(paramKey)
	} else {
		return ""
	}
}

func jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.RequestURI, "/healthz") {
			return
		}

		h := AuthHeader{}
		if err := c.ShouldBindHeader(&h); err != nil || h.Authorization == "" {
			log.Debugf("No Authorization header but JWT checking is enabled, returning 401")
			jwtRequestsMetric.WithLabelValues("false", "No Authorization header").Inc()
			c.JSON(401, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		jwtToken := strings.TrimSpace(strings.Replace(h.Authorization, "Bearer", "", 1))

		token, err := jwt.ParseWithClaims(jwtToken, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtRsaPubKey, nil
		})

		if err != nil {
			log.Debugf("JWT token couldn't be parsed: %v", err)
			if ve, ok := err.(*jwt.ValidationError); ok {
				if ve.Errors&(jwt.ValidationErrorExpired) != 0 {
					jwtRequestsMetric.WithLabelValues("false", "JWT expired").Inc()
					c.JSON(401, gin.H{"error": "JWT token expired"})
					c.Abort()
					return
				} else if ve.Errors&(jwt.ValidationErrorMalformed) != 0 {
					jwtRequestsMetric.WithLabelValues("false", "JWT malformed").Inc()
					c.JSON(401, gin.H{"error": "JWT token malformed"})
					c.Abort()
					return
				} else if ve.Errors&(jwt.ValidationErrorUnverifiable|jwt.ValidationErrorSignatureInvalid) != 0 {
					jwtRequestsMetric.WithLabelValues("false", "JWT signature incorrect").Inc()
					c.JSON(401, gin.H{"error": "JWT token signature incorrect"})
					c.Abort()
					return
				}
			}
		}
		if token == nil {
			log.Debugf("JWT token failed to be parsed with given claims")
			jwtRequestsMetric.WithLabelValues("false", "JWT claims invalid").Inc()
			c.JSON(401, gin.H{"error": "JWT token invalid"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid {

			if jwtIssuerFlag != "" && strings.TrimSpace(claims.StandardClaims.Issuer) != jwtIssuerFlag {
				log.Debugf("JWT token issuer claim doesn't match provided -jwt-issuer value")
				jwtRequestsMetric.WithLabelValues("false", "JWT token issuer claim does not match").Inc()
				c.JSON(401, gin.H{"error": "JWT token issuer is not valid"})
				c.Abort()
				return
			}

			if jwtAudienceFlag != "" && strings.TrimSpace(claims.StandardClaims.Audience) != jwtAudienceFlag {
				log.Debugf("JWT token audience claim doesn't match provided -jwt-audience value")
				jwtRequestsMetric.WithLabelValues("false", "JWT token audience claim does not match").Inc()
				c.JSON(401, gin.H{"error": "JWT token audience is not valid"})
				c.Abort()
				return
			}

			if !validActionForHttpMethod(claims.Action, c.Request.Method) {
				log.Debugf("Got valid JWT token, but action allow doesn't match request (action %s != method %s)", claims.Action, c.Request.Method)
				jwtRequestsMetric.WithLabelValues("false", "JWT action does not match method").Inc()
				c.JSON(401, gin.H{"error": "JWT token action allow doesn't match request method"})
				c.Abort()
				return
			}

			keyParam := getRequestParam(c, "key")
			bucketParam := getRequestParam(c, "bucket")

			if keyParam == "" {
				keyParam = getRequestParam(c, "prefix")
			}

			if keyParam == "" {
				keyParam = getRequestParam(c, "path")
			}

			// Compare request bucket and key params with JWT bucket and key params
			if keyParam != "" && strings.TrimSpace(claims.Prefix) != "" && !strings.HasPrefix(keyParam, strings.TrimSpace(claims.Prefix)) {
				log.Debugf("JWT token prefix does not match URL object (prefix %s != object %s)", claims.Prefix, strings.TrimSpace(keyParam))
				jwtRequestsMetric.WithLabelValues("false", "JWT token prefix does not match URL object").Inc()
				c.JSON(401, gin.H{"error": "JWT token prefix does not match URL object"})
				c.Abort()
				return
			}

			if bucketParam != "" && strings.TrimSpace(claims.Bucket) != "" && strings.TrimSpace(claims.Bucket) != bucketParam {
				log.Debugf("JWT token bucket does not match URL bucket (jwt bucket %s != URL bucket %s", claims.Bucket, strings.TrimSpace(bucketParam))
				jwtRequestsMetric.WithLabelValues("false", "JWT token bucket does not match URL bucket").Inc()
				c.JSON(401, gin.H{"error": "JWT token bucket does not match URL bucket"})
				c.Abort()
				return
			}

			log.Debugf("Got valid JWT token, exiting JWT middleware")
			jwtRequestsMetric.WithLabelValues("true", "").Inc()
		} else {
			jwtRequestsMetric.WithLabelValues("false", "JWT claims invalid").Inc()
			c.JSON(401, gin.H{"error": "JWT token invalid"})
			c.Abort()
		}
	}
}

func validActionForHttpMethod(action string, method string) bool {
	switch action {
	case "READ":
		return (method == "GET" || method == "HEAD")
	case "WRITE":
		return (method == "POST" || method == "PUT")
	case "DELETE":
		return method == "DELETE"
	}

	return false
}
