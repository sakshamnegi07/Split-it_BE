package controller

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
	"gopkg.in/gomail.v2"
	"gorm.io/gorm"
)

func AddExpense(ctx *gin.Context) {
	var expenseData models.Expense
	if err := ctx.ShouldBindJSON(&expenseData); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := database.DB.Begin()

	// Save expense to database
	if err := tx.Create(&expenseData).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create expense"})
		return
	}

	// Get group members
	var members []Member
	if err := tx.Where("group_id = ?", expenseData.GroupID).Find(&members).Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch group members"})
		return
	}

	// Calculate split amount
	splitAmount := expenseData.Amount / float64(len(members))
	splitAmount = math.Floor(splitAmount*100) / 100

	// Update balances
	for _, member := range members {
		if member.UserID == expenseData.PaidBy {
			continue // Skip the payer
		}

		var balance models.Balance
		result := tx.Where("group_id = ? AND lender = ? AND borrower = ?", expenseData.GroupID, expenseData.PaidBy, member.UserID).First(&balance)

		if result.Error == gorm.ErrRecordNotFound {
			balance = models.Balance{
				GroupID:  expenseData.GroupID,
				Lender:   expenseData.PaidBy,
				Borrower: member.UserID,
				Amount:   splitAmount,
			}
			if err := tx.Create(&balance).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create balance"})
				return
			}
		} else if result.Error != nil {
			tx.Rollback()
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance"})
			return
		} else {
			newAmount := balance.Amount + splitAmount
			if err := tx.Model(&balance).Where("balance_id = ?", balance.BalanceId).Update("amount", newAmount).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
				return
			}
		}

		// Check if reverse balance record exists
		var reverseBalance models.Balance
		reverseResult := tx.Where("group_id = ? AND lender = ? AND borrower = ?", expenseData.GroupID, member.UserID, expenseData.PaidBy).First(&reverseBalance)

		if reverseResult.Error == gorm.ErrRecordNotFound {
			reverseBalance = models.Balance{
				GroupID:  expenseData.GroupID,
				Lender:   member.UserID,
				Borrower: expenseData.PaidBy,
				Amount:   -splitAmount,
			}
			if err := tx.Create(&reverseBalance).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reverse balance"})
				return
			}
		} else if reverseResult.Error != nil {
			tx.Rollback()
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch reverse balance"})
			return
		} else {
			newAmount := reverseBalance.Amount - splitAmount
			if err := tx.Model(&reverseBalance).Where("balance_id = ?", reverseBalance.BalanceId).Update("amount", newAmount).Error; err != nil {
				tx.Rollback()
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reverse balance"})
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	go func(members []Member, expenseData models.Expense) {
		sendEmailForExpenseCreation(members, expenseData.PaidBy, expenseData.Amount)
	}(members, expenseData)

	ctx.JSON(http.StatusCreated, expenseData)
}

func GetExpensesByGroupID(ctx *gin.Context) {
	groupID := ctx.Param("group_id")

	var expenses []models.ExpenseWithUser
	if err := database.DB.Table("expenses").
		Select("expenses.*, users.username").
		Joins("LEFT JOIN users ON expenses.paid_by = users.id").
		Where("expenses.group_id = ?", groupID).
		Order("expenses.created_at DESC").
		Find(&expenses).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, expenses)
}

func GetOverallGroupBalance(ctx *gin.Context) {
	groupID := ctx.Param("group_id")
	currentUserID, exists := ctx.Get("userId")

	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	var totalBalance float64

	err := database.DB.Model(&models.Balance{}).
		Select("COALESCE(SUM(amount), 0) as total_balance").
		Where("lender = ? AND group_id = ?", currentUserID, groupID).
		Scan(&totalBalance).Error

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balances: " + err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"total_balance": totalBalance})
}

func sendEmailForExpenseCreation(members []Member, currentUserId uint, amount float64) {
	currentUser, err := fetchUser(currentUserId)
	if err != nil {
		log.Printf("Error fetching email for current user")
		return
	}
	for _, member := range members {
		if member.UserID != currentUserId {
			toUser, err := fetchUser(member.UserID)
			if err != nil {
				log.Printf("Error fetching email for user ID %d: %v\n", member.UserID, err)
				continue
			}
			group, err := fetchGroup(member.GroupID)
			if err != nil {
				log.Printf("Error fetching group for current users")
				return
			}

			m := gomail.NewMessage()
			m.SetHeader("From", "splitapplicationcustom@gmail.com")
			m.SetHeader("To", toUser.Email)
			m.SetHeader("Subject", "New expense added")
			m.SetBody("text/html",
				"<p>Hello <b>"+toUser.Username+"</b>,</p>"+
					"<p><b>"+currentUser.Username+"</b> has added an expense of <b>Rs."+fmt.Sprintf("%.2f", amount)+"</b> in the group <b>"+group.GroupName+"</b> in Split-it app.</p>"+
					"<p>Best regards,<br>Split-it Team</p>")

			mailPort, err := strconv.Atoi(os.Getenv("MAIL_PORT"))
			if err != nil {
				return
			}
			d := gomail.NewDialer(os.Getenv("MAIL_HOST"), mailPort, os.Getenv("MAIL_ADDRESS"), os.Getenv("MAIL_PASS"))

			if err := d.DialAndSend(m); err != nil {
				return
			}
			return
		}
	}
}

func fetchUser(userID uint) (models.User, error) {
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

func fetchGroup(groupID uint) (Group, error) {
	var group Group
	if err := database.DB.Where("group_id = ?", groupID).First(&group).Error; err != nil {
		return group, err
	}
	return group, nil
}
