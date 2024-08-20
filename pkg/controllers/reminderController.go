package controller

import (
	"net/http"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
	"gopkg.in/gomail.v2"
)

func SettlementReminder(ctx *gin.Context) {
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

	var currentUser, inputUser User
	if err := database.DB.First(&currentUser, userID).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Current user not found"})
		return
	}
	if err := database.DB.First(&inputUser, request.Borrower).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Input user not found"})
		return
	}

	reminder := models.Reminder{
		SentBy: currentUser.ID,
		SentTo: inputUser.ID,
		Amount: request.Amount,
	}
	if err := database.DB.Create(&reminder).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reminder"})
		return
	}

	if err := sendReminderEmail(currentUser, inputUser, request.Amount); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Reminder email sent successfully"})
}

func sendReminderEmail(fromUser, toUser User, amount float64) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "splitapplicationcustom@gmail.com")
	m.SetHeader("To", toUser.Email)
	m.SetHeader("Subject", "Payment Reminder")
	m.SetBody("text/html",
		"<p>Hello <b>"+toUser.Username+"</b>,</p>"+
			"<p><b>"+fromUser.Username+"</b> has requested you to pay your share which is <b>Rs."+fmt.Sprintf("%.2f", amount)+"</b> in the Split-it app.</p>"+
			"<p>Best regards,<br>Split-it Team</p>")

	d := gomail.NewDialer("smtp.gmail.com", 587, "splitapplicationcustom@gmail.com", "yultnhjwafmumazt")

	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
