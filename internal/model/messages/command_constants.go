package messages

const (
	NoCommand int = iota
	StartCmd
	ResetCmd
	AddCategoryCmd
	AddExpenceCmd
	GetReportCmd
	ChangeCurrency
	SetMonthLimit
	ResetMonthLimit
	GetHelpCmd
)

type CommandInfo struct {
	Command     string
	Description string
	Format      string
}

var CommandNameMap = map[int]CommandInfo{
	StartCmd:        {"start", "Start bot", ""},
	ResetCmd:        {"reset", "Reset all expence data", ""},
	AddCategoryCmd:  {"add_category", "Add new category", "<category>"},
	AddExpenceCmd:   {"add_expence", "Add new expence", "<category> <total> <date>"},
	GetReportCmd:    {"report", "Get expence report by day/month/year", "?<day/month/year>"},
	ChangeCurrency:  {"currency", "Change currency", "<USD/CNY/EUR/RUB>"},
	SetMonthLimit:   {"set_limit", "Set month limit", "<total>"},
	ResetMonthLimit: {"reset_limit", "Reset month limit", ""},
	GetHelpCmd:      {"help", "Get help", ""},
}
