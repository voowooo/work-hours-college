package models

import (
	"time"
)

type User struct {
	ID         string    `json:"id"`
	TelegramID int64     `json:"telegramId"`
	Username   *string   `json:"username"`
	CreatedAt  time.Time `json:"createdAt"`
}

type WorkEntry struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	Date        string    `json:"date"`
	ArrivedAt   *string   `json:"arrivedAt"`
	LeftAt      *string   `json:"leftAt"`
	HoursWorked float64   `json:"hoursWorked"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type MonthStats struct {
	Month      string      `json:"month"`
	TotalHours float64     `json:"totalHours"`
	Entries    []WorkEntry `json:"entries"`
}

type UpsertWorkEntryInput struct {
	Date      string  `json:"date"`
	ArrivedAt *string `json:"arrivedAt"`
	LeftAt    *string `json:"leftAt"`
}
