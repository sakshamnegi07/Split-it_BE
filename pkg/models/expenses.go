package models

import "time"

type Expense struct {
	PaidBy      uint    `json:"paid_by"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	GroupID     uint    `json:"group_id"`
}

type ExpenseWithUser struct {
	Expense
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}
