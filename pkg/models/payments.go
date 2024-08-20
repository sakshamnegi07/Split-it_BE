package models

import "time"

type Payments struct {
	ID     uint      `gorm:"column:payment_id"`
	PaidBy uint      `json:"paid_by"`
	PaidTo uint      `json:"paid_to"`
	Amount float64   `json:"amount"`
	Date   time.Time `gorm:"column:paid_at"`
}
