package helpers

import "time"

func StringToDate(stringDate string) (time.Time, error) {
	rv, err := time.Parse("02/01/2006", stringDate)
	return rv, err
}
