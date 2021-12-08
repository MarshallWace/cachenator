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
	Url    string `json:"url"`
	jwt.StandardClaims
}

type AuthHeader struct {
	Authorization string `header:"Authorization"`
}

func jwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := AuthHeader{}

		if err := c.ShouldBindHeader(&h); err != nil || h.Authorization == "" {
			c.JSON(401, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		jwtToken := strings.TrimSpace(strings.ReplaceAll(h.Authorization, "Bearer", ""))

		token, err := jwt.ParseWithClaims(jwtToken, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
			return jwtRsaPubKey, nil
		})
		if token == nil {
			c.JSON(401, gin.H{"error": "JWT token invalid"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*JwtClaims); ok && token.Valid && err == nil {
			log.Infof("Bearer token: %v", claims.StandardClaims.Subject)
			c.Next()
		} else {
			c.JSON(401, gin.H{"error": "JWT token invalid"})
			c.Abort()
		}
	}
}
