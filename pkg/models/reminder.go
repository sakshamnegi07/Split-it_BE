package models

type Reminder struct {
	ID     uint    `gorm:"primaryKey"`
	SentBy uint    `json:"sent_by"`
	SentTo uint    `json:"sent_to"`
	Amount float64 `json:"amount"`
}
