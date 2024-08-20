package controller

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	"github.com/sakshamnegi07/split-it/pkg/models"
)

func DownloadPaymentsCSV(ctx *gin.Context) {
	userID, exists := ctx.Get("userId")
	if !exists {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
		return
	}

	var payments []models.Payments
	result := database.DB.Where("paid_by = ? OR paid_to = ?", userID, userID).Find(&payments)
	if result.Error != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	userIDs := getUserIDs(payments)
	users, err := getUsersByIds(userIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Header("Content-Disposition", "attachment; filename=payments_report.csv")
	ctx.Header("Content-Type", "text/csv")
	err = generateCSV(ctx.Writer, payments, users)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func getUserIDs(payments []models.Payments) []int {
	uniqueIDs := make(map[int]struct{})
	for _, payment := range payments {
		uniqueIDs[int(payment.PaidBy)] = struct{}{}
		uniqueIDs[int(payment.PaidTo)] = struct{}{}
	}

	ids := make([]int, 0, len(uniqueIDs))
	for id := range uniqueIDs {
		ids = append(ids, id)
	}
	return ids
}

func getUsersByIds(ids []int) (map[int]models.User, error) {
	var users []models.User
	result := database.DB.Where("id IN (?)", ids).Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}

	userMap := make(map[int]models.User)
	for _, user := range users {
		userMap[int(user.ID)] = user
	}
	return userMap, nil
}

func generateCSV(w http.ResponseWriter, payments []models.Payments, users map[int]models.User) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"ID", "Amount", "Date", "Paid By", "Paid To"})

	for _, payment := range payments {
		paidByUser := users[int(payment.PaidBy)]
		paidToUser := users[int(payment.PaidTo)]

		row := []string{
			strconv.Itoa(int(payment.ID)),
			fmt.Sprintf("%.2f", payment.Amount),
			payment.Date.Format("2006-01-02"),
			paidByUser.Username,
			paidToUser.Username,
		}
		writer.Write(row)
	}
	return nil
}
