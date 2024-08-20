package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
	"gorm.io/gorm"
)

type Group struct {
	ID               uint   `json:"group_id" gorm:"primaryKey;autoIncrement;column:group_id"`
	GroupName        string `json:"group_name"`
	GroupDescription string `json:"group_description"`
	CreatedBy        uint   `json:"created_by"`
}

type Member struct {
	UserID    uint           `json:"user_id"`
	GroupID   uint           `json:"group_id"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type MemberWithBalance struct {
	User
	Amount float64 `json:"amount"`
}

type GroupWithBalance struct {
	Group
	Amount float64 `json:"amount"`
}

func CreateGroup(ctx *gin.Context) {
	var group Group
	if err := ctx.ShouldBindJSON(&group); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {

		if err := tx.Create(&group).Error; err != nil {
			return err
		}

		member := Member{
			UserID:  group.CreatedBy,
			GroupID: group.ID,
		}

		if err := tx.Create(&member).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, group)
}

func GetUserGroups(ctx *gin.Context) {
	userID := ctx.Param("id")
	var groups []Group
	err := database.DB.Table("groups").
		Select("*").
		Joins("JOIN members ON members.group_id = groups.group_id").
		Where("members.user_id = ?", userID).
		Order("groups.created_at DESC").
		Find(&groups).Error

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var groupWithBalance []GroupWithBalance
	for _, group := range groups {

		var totalBalance float64

		err := database.DB.Model(&models.Balance{}).
			Select("COALESCE(SUM(amount), 0.0) as total_balance").
			Where("lender = ? AND group_id = ?", userID, group.ID).
			Scan(&totalBalance).Error

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance: " + err.Error()})
			return
		}

		groupWithBalance = append(groupWithBalance, GroupWithBalance{
			Group:  group,
			Amount: totalBalance,
		})
	}

	ctx.JSON(http.StatusOK, groupWithBalance)
}

func GetGroupMembers(ctx *gin.Context) {
	groupID := ctx.Param("group_id")
	var members []Member
	var users []User

	currentUserID, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	if err := database.DB.Where("group_id = ?", groupID).Find(&members).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var userIDs []uint
	for _, member := range members {
		userIDs = append(userIDs, member.UserID)
	}

	if err := database.DB.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user details: " + err.Error()})
		return
	}

	var membersWithAmount []MemberWithBalance
	for _, user := range users {
		var amount float64
		if user.ID != currentUserID {
			err := database.DB.Table("balances").
				Select("COALESCE(SUM(amount), 0)").
				Where("group_id = ? AND lender = ? AND borrower = ?", groupID, currentUserID, user.ID).
				Scan(&amount).Error
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate balance: " + err.Error()})
				return
			}
		}

		membersWithAmount = append(membersWithAmount, MemberWithBalance{
			User:   user,
			Amount: amount,
		})
	}

	ctx.JSON(http.StatusOK, membersWithAmount)
}

func AddGroupMember(ctx *gin.Context) {
	groupIDStr := ctx.Param("group_id")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	var member Member
	if err := ctx.ShouldBindJSON(&member); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	member.GroupID = uint(groupID)
	var existingMember Member
	if err := database.DB.Where("group_id = ? AND user_id = ?", member.GroupID, member.UserID).First(&existingMember).Error; err == nil {
		ctx.JSON(http.StatusConflict, gin.H{"Error": "Member already in group"})
		return
	}

	if err := database.DB.Create(&member).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, member)
}

func RemoveGroupMember(ctx *gin.Context) {
	groupIDStr := ctx.Param("group_id")
	groupID, err := strconv.ParseUint(groupIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid group_id"})
		return
	}

	userIDStr := ctx.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var balances []models.Balance
	if err := database.DB.Where("group_id = ? AND borrower = ?", uint(groupID), uint(userID)).Find(&balances).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check balances"})
		return
	}

	for _, balance := range balances {
		if balance.Amount != 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "Member has non-zero balance with other group members!"})
			return
		}
	}

	if err := database.DB.Where("group_id = ? AND user_id = ?", uint(groupID), uint(userID)).Delete(&Member{}).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member removed from group!"})
}
