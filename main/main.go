package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sakshamnegi07/split-it/main/database"
	controller "github.com/sakshamnegi07/split-it/pkg/controllers"
)

func main() {
	r := gin.Default()
	database.Connect()

	// Auth routes
	r.POST("/register", controller.Register)
	r.POST("/login", controller.Login)

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

	r.Run(":9090")
}
