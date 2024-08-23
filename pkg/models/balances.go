package models

import "time"

type Balance struct {
	BalanceId uint
	GroupID   uint
	Lender    uint
	Borrower  uint
	Amount    float64 `json:"amount"`
}

type BalanceSummary struct {
	BorrowerID    uint    `json:"borrower_id"`
	BorrowerName  string  `json:"borrower_name"`
	BorrowerEmail string  `json:"borrower_email"`
	TotalAmount   float64 `json:"total_amount"`
}

type Payment struct {
	PaidBy uint      `json:"paid_by"`
	PaidTo uint      `json:"paid_to"`
	Amount float64   `json:"amount"`
	PaidAt time.Time `json:"paid_at" gorm:"default:CURRENT_TIMESTAMP"`
}
