package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
	"gorm.io/gorm"
)

func GetUserByEmail(ctx *gin.Context) {
	email := ctx.Query("email")

	user := models.User{
		Email: email,
	}

	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	response := models.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	}

	ctx.JSON(http.StatusOK, response)
}

func GetUserDetails(ctx *gin.Context) {
	currentUserID, exists := ctx.Get("userId")

	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	var user User
	result := database.DB.First(&user, currentUserID)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func Logout(ctx *gin.Context) {
	currentUserID, exists := ctx.Get("userId")

	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	var user User
	result := database.DB.First(&user, currentUserID)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		}
		return
	}

	ctx.JSON(http.StatusOK, user)
}
