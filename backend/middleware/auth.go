package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"stratus/config"
	"stratus/database"
	"stratus/models"
)

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if !user.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "User account is disabled"})
			c.Abort()
			return
		}

		c.Set("user", &user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			c.Abort()
			return
		}

		u := user.(*models.User)
		if !u.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) *models.User {
	user, exists := c.Get("user")
	if !exists {
		return nil
	}
	return user.(*models.User)
}

func BasicAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		const basicAuthPrefix = "Basic "
		if !strings.HasPrefix(authHeader, basicAuthPrefix) {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		payload, err := base64.StdEncoding.DecodeString(authHeader[len(basicAuthPrefix):])
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		username := pair[0]
		password := pair[1]

		var user models.User
		if err := database.DB.Where("email = ?", username).First(&user).Error; err != nil {
			if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
				c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
				c.Status(http.StatusUnauthorized)
				c.Abort()
				return
			}
		}

		if !VerifyPassword(user.PasswordHash, password) {
			c.Header("WWW-Authenticate", `Basic realm="WebDAV"`)
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}

		if !user.IsActive {
			c.Status(http.StatusForbidden)
			c.Abort()
			return
		}

		c.Set("user", &user)
		c.Set("userID", user.ID)
		c.Next()
	}
}

func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
