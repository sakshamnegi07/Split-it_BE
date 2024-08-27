package controller

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
	"gopkg.in/gomail.v2"
)

func GetFriendsWithBalances(ctx *gin.Context) {
	userID, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}
	var user User

	balanceSummaries := []models.BalanceSummary{}

	// Find the user
	if err := database.DB.First(&user, userID).Error; err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if err := database.DB.Model(&models.Balance{}).
		Select("users.id as borrower_id, users.username as borrower_name, users.email as borrower_email, SUM(balances.amount) as total_amount").
		Joins("left join users on users.id = balances.borrower").
		Where("balances.lender = ?", userID).
		Group("balances.borrower, users.id, users.username, users.email").
		Scan(&balanceSummaries).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving balances"})
		return
	}

	var totalAmountBorrowed float64
	for _, summary := range balanceSummaries {
		totalAmountBorrowed += summary.TotalAmount
	}

	totalAmountBorrowed = float64(int(totalAmountBorrowed*100)) / 100

	ctx.JSON(http.StatusOK, gin.H{
		"user_id":      user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"total_amount": totalAmountBorrowed,
		"balances":     balanceSummaries,
	})
}

func SettleMoney(ctx *gin.Context) {
	userID, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	var request struct {
		Borrower uint    `json:"borrower"`
		Amount   float64 `json:"amount"`
	}
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	database.DB.Model(&models.Balance{}).
		Where("(lender = ? AND borrower = ?) OR (lender = ? AND borrower = ?)", userID, request.Borrower, request.Borrower, userID).
		Update("amount", 0.00)

	payment := models.Payment{
		PaidBy: userID.(uint),
		PaidTo: request.Borrower,
		Amount: math.Abs(request.Amount),
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating user"})
		return
	}

	var currentUser, lenderUser User
	if err := database.DB.First(&currentUser, userID).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Input user not found"})
		return
	}
	if err := database.DB.First(&lenderUser, request.Borrower).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Input user not found"})
		return
	}

	go sendSettledPaymentMail(currentUser, lenderUser, request.Amount)

	ctx.JSON(http.StatusOK, gin.H{"message": "Balances settled successfully"})
}

func GetPaymentHistory(ctx *gin.Context) {
	fmt.Print("checking")
	userID, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	var payments []struct {
		models.Payment
		PaidByUsername string `json:"paid_by_username"`
		PaidByEmail    string `json:"paid_by_email"`
		PaidToUsername string `json:"paid_to_username"`
		PaidToEmail    string `json:"paid_to_email"`
	}

	err := database.DB.Table("payments").
		Select("payments.*, "+
			"u1.username AS paid_by_username, u1.email AS paid_by_email, "+
			"u2.username AS paid_to_username, u2.email AS paid_to_email").
		Joins("LEFT JOIN users u1 ON payments.paid_by = u1.id").
		Joins("LEFT JOIN users u2 ON payments.paid_to = u2.id").
		Where("payments.paid_by = ? OR payments.paid_to = ?", userID, userID).
		Find(&payments).Error

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, payments)
}

func sendSettledPaymentMail(fromUser, toUser User, amount float64) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "splitapplicationcustom@gmail.com")
	m.SetHeader("To", toUser.Email)
	m.SetHeader("Subject", "Payment Reminder")
	m.SetBody("text/html",
		"<p>Hello <b>"+toUser.Username+"</b>,</p>"+
			"<p><b>"+fromUser.Username+"</b> has paid your share which was <b>Rs."+fmt.Sprintf("%.2f", amount)+"</b> in the Split-it app.</p>"+
			"<p>Best regards,<br>Split-it Team</p>")

	mailPort, err := strconv.Atoi(os.Getenv("MAIL_PORT"))
	if err != nil {
		return err
	}
	d := gomail.NewDialer(os.Getenv("MAIL_HOST"), mailPort, os.Getenv("MAIL_ADDRESS"), os.Getenv("MAIL_PASS"))

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
