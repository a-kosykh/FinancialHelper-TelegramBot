package domain

type User struct {
	UserID            int64
	BaseCurrencyID    uint64
	DefaultMonthLimit int64
	CurrentMonthLimit int64
}
