package auth

import (
	"ai-collab-hub/internal/config"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

func HashPass(p string) string { h, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost); return string(h) }
func CheckPass(p, h string) bool { return bcrypt.CompareHashAndPassword([]byte(h), []byte(p)) == nil }

func GenToken(id uint, role string) string {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"id": id, "role": role, "exp": time.Now().Add(72*time.Hour).Unix()})
	s, _ := t.SignedString([]byte(config.Cfg.Server.JWTSecret))
	return s
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if len(tokenStr) < 7 { c.AbortWithStatusJSON(401, gin.H{"error": "Unauthorized"}); return }
		tokenStr = tokenStr[7:]
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tokenStr, &claims, func(t *jwt.Token) (interface{}, error) { return []byte(config.Cfg.Server.JWTSecret), nil })
		if err != nil { c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"}); return }
		c.Set("user", claims)
		if role, ok := claims["role"].(string); ok {
			c.Set("role", role)
		}
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" { c.AbortWithStatusJSON(403, gin.H{"error": "Admin only"}); return }
		c.Next()
	}
}
