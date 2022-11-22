package userdata

import "gitlab.ozon.dev/akosykh114/telegram-bot/internal/model/expences"

type UserData struct {
	ExpencesMap  map[string][]expences.Expence
	BaseCurrency string
}
