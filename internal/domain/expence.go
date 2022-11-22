package domain

import "time"

type Expence struct {
	ID           int64
	UserID       int64
	CategoryID   int64
	CategoryName string
	Timestamp    time.Time
	Total        int64
}
