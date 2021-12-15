package main

import (
	"crypto/rsa"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
)

var (
	jwtRsaPubKey *rsa.PublicKey
)

type JwtClaims struct {
	Action string `json:"action"`
	// TODO: Handle bucket+key validation
	// Bucket string `json:"bucket"`
	// Key string `json:"key"`
	jwt.StandardClaims
}

type AuthHeader struct {
	Authorization string `header:"Authorization"`
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

		jwtToken := strings.TrimSpace(strings.ReplaceAll(h.Authorization, "Bearer", ""))

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
			// TODO: Add JWT standard claims Issuer+Audience validation
			if !validActionForHttpMethod(claims.Action, c.Request.Method) {
				log.Debugf("Got valid JWT token, but action allow doesn't match request (action %s != method %s)", claims.Action, c.Request.Method)
				jwtRequestsMetric.WithLabelValues("false", "JWT action does not match method").Inc()
				c.JSON(401, gin.H{"error": "JWT token action allow doesn't match request method"})
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
