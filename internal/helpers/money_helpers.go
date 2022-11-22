package helpers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

var ErrInvalidAmount = errors.New("invalid amount")

// 1,000,500.10 -> 1000500.10 || 1 000 500.100 -> 1000500.100
var regexpNoNumber = regexp.MustCompile(`[^\d\.]`)

// ConvertStringAmountToSub - convert amount to Sub
func ConvertStringAmountToSub(amount string) (int64, error) {
	v, err := strconv.ParseFloat(regexpNoNumber.ReplaceAllString(amount, ""), 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	return ConvertFloat64AmountToSub(v)
}

// ConvertFloat64AmountToSub - convert amount to Sub
func ConvertFloat64AmountToSub(amount float64) (int64, error) {
	return int64(amount * 100), nil
}

// ConvertSubToAmount - convert Sub to amount
func ConvertSubToAmount(Sub int64) string {
	amount := fmt.Sprintf("%d", Sub)
	if len(amount) < 3 {
		return fmt.Sprintf("0.%s", amount)
	}
	return fmt.Sprintf("%s.%s", amount[:len(amount)-2], amount[len(amount)-2:])
}
