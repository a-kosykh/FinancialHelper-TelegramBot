package domain

import "time"

type ReportRequest struct {
	UserID    int64
	Timestamp time.Time
}
