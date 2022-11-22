package helpers

import "time"

func GetNowDateTimeLoc() (int, time.Month, int, *time.Location) {
	now := time.Now()
	currentYear, currentMonth, currentDay := now.Date()
	currentLocation := now.Location()
	return currentYear, currentMonth, currentDay, currentLocation
}

func GetStartOfCurrentYear() time.Time {
	currentYear, _, _, loc := GetNowDateTimeLoc()
	return time.Date(currentYear, 1, 1, 0, 0, 0, 0, loc)
}

func GetStartOfCurrentMonth() time.Time {
	year, month, _, loc := GetNowDateTimeLoc()
	return time.Date(year, month, 1, 0, 0, 0, 0, loc)
}

func GetStartOfCurrentDay() time.Time {
	year, month, day, loc := GetNowDateTimeLoc()
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}
