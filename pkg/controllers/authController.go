package controller

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("secret_key")

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func Register(ctx *gin.Context) {
	var creds models.Credentials
	if err := ctx.BindJSON(&creds); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	hashedPassword, err := hashPassword(creds.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error hashing password"})
		return
	}

	user := models.User{
		Username: creds.Username,
		Email:    creds.Email,
		Password: hashedPassword,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"message": "User created successfully"})
}

func Login(ctx *gin.Context) {
	var creds models.LoginCredentials
	if err := ctx.BindJSON(&creds); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", creds.Email).First(&user).Error; err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	if !checkPasswordHash(creds.Password, user.Password) {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.Claims{
		Id:    user.ID,
		Email: creds.Email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating token"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"token": tokenString, "userId": user.ID, "username": user.Username})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tokenString := ctx.GetHeader("Authorization")
		if tokenString == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization token required"})
			ctx.Abort()
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token signature"})
				ctx.Abort()
				return
			}
		}

		if !token.Valid {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			ctx.Abort()
			return
		}

		ctx.Set("email", claims.Email)
		ctx.Set("userId", claims.Id)
		ctx.Next()
	}
}
