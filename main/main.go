package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sakshamnegi07/split-it/main/database"
	controller "github.com/sakshamnegi07/split-it/pkg/controllers"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		fmt.Println("Error loading .env file", err)
	}

	r := gin.Default()
	database.Connect()

	// Auth routes
	r.POST("/register", controller.Register)
	r.POST("/login", controller.Login)
	r.GET("/validate-user", controller.AuthMiddleware(), controller.ValidateUser)

	// User routes
	r.GET("/get-user/", controller.AuthMiddleware(), controller.GetUserByEmail)
	r.GET("/user/details", controller.AuthMiddleware(), controller.GetUserDetails)

	// Group routes
	r.POST("/groups", controller.AuthMiddleware(), controller.CreateGroup)
	r.GET("/groups/:id", controller.AuthMiddleware(), controller.GetUserGroups)

	// adding and removing group members
	r.GET("/groups/members/:group_id", controller.AuthMiddleware(), controller.GetGroupMembers)
	r.POST("/groups/members/:group_id", controller.AuthMiddleware(), controller.AddGroupMember)
	r.DELETE("/groups/members/:group_id/delete/:user_id", controller.AuthMiddleware(), controller.RemoveGroupMember)

	// expenses
	r.POST("/expenses/add-expense", controller.AuthMiddleware(), controller.AddExpense)
	r.GET("/expenses/:group_id", controller.AuthMiddleware(), controller.GetExpensesByGroupID)
	r.GET("/expenses/overall-expense/:group_id", controller.AuthMiddleware(), controller.GetOverallGroupBalance)

	// settlements
	r.GET("/balances", controller.AuthMiddleware(), controller.GetFriendsWithBalances)
	r.POST("/settle", controller.AuthMiddleware(), controller.SettleMoney)

	//payments-history
	r.GET("/payments", controller.AuthMiddleware(), controller.GetPaymentHistory)

	//reminders
	r.POST("/remind-user", controller.AuthMiddleware(), controller.SettlementReminder)

	//downloading report
	r.GET("/download/payments/csv", controller.AuthMiddleware(), controller.DownloadPaymentsCSV)

	//getting PORT from env file
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
