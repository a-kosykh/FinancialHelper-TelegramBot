package messages

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/common"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/helpers"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	metr "gitlab.ozon.dev/akosykh114/telegram-bot/internal/metrics"
	"go.uber.org/zap"
)

type MessageSender interface {
	SendMessage(text string, userID int64) error
}

type storageInterface interface {
	IsUserAdded(ctx context.Context, userID int64) bool
	AddUser(ctx context.Context, userID int64) bool
	ResetUser(ctx context.Context, userID int64) bool
	ChangeCurrency(ctx context.Context, userID int64, currency string) bool
	IsCategoryExists(ctx context.Context, userID int64, cat string) bool
	AddCategory(ctx context.Context, userID int64, cat string) bool
	AddExpence(ctx context.Context, userID int64, cat string, total int64, date time.Time) error
	GetExpencesMap(ctx context.Context, userID int64, limitTs time.Time) map[string]int64
	SetUserLimit(ctx context.Context, userID int64, total int64) error
	ResetUserLimit(ctx context.Context, userID int64) error
}

type ReportGetter interface {
	//GetReportChan() chan map[string]int64
}

type Model struct {
	tgClient MessageSender
	storage  storageInterface
}

func New(
	tgClient MessageSender,
	storage storageInterface) *Model {
	return &Model{
		tgClient: tgClient,
		storage:  storage,
	}
}

type Message struct {
	UserID int64
}

type PlainTextMessage struct {
	Message Message
	Text    string
}

type CommandMessage struct {
	Message          Message
	CommandName      string
	CommandArguments string
}

var errWrongCommandFormat = fmt.Errorf(fmt.Sprintf("wrong command format - use '/%s'", CommandNameMap[GetHelpCmd].Command))
var errCategoryNotFound = fmt.Errorf(fmt.Sprintf("category was not found - use '/%s %s'", CommandNameMap[AddCategoryCmd].Command, CommandNameMap[AddCategoryCmd].Format))
var errDateWrongFormat = fmt.Errorf("wrong date format - right: dd/mm/yyyy")
var errLimitIsTooSmall = fmt.Errorf("limit is too small")
var errResetLimit = fmt.Errorf("error reseting limit")
var errServer = fmt.Errorf("server error")

func (s *Model) IncomingPlainTextMessage(msg PlainTextMessage) error {
	return s.tgClient.SendMessage(s.Help(), msg.Message.UserID)
}

func (s *Model) IncomingCommandMessage(ctx context.Context, msg CommandMessage) error {
	var err error
	var answer string

	span, ctx := opentracing.StartSpanFromContext(ctx, "incoming_command_process")
	defer span.Finish()

	span.LogKV(
		"command", msg.CommandName,
		"argument", msg.CommandArguments,
	)

	if !s.storage.IsUserAdded(ctx, msg.Message.UserID) {
		if _, err = s.addUser(ctx, msg.Message.UserID); err != nil {
			logger.Error("adding user error", zap.Error(err))
		}
	}

	startTime := time.Now()

	switch msg.CommandName {
	case CommandNameMap[StartCmd].Command:
		answer = s.welcomeMessage()
	case CommandNameMap[ResetCmd].Command:
		answer, err = s.resetUser(ctx, msg.Message.UserID)
	case CommandNameMap[AddCategoryCmd].Command:
		answer, err = s.AddCategory(ctx, msg.Message.UserID, msg.CommandArguments)
	case CommandNameMap[AddExpenceCmd].Command:
		answer, err = s.AddExpence(ctx, msg.Message.UserID, msg.CommandArguments)
	case CommandNameMap[GetReportCmd].Command:
		answer, err = s.GetReport(ctx, msg.Message.UserID, msg.CommandArguments)
	case CommandNameMap[ChangeCurrency].Command:
		answer, err = s.ChangeCurrency(ctx, msg.Message.UserID, msg.CommandArguments)
	case CommandNameMap[SetMonthLimit].Command:
		answer, err = s.SetUserLimit(ctx, msg.Message.UserID, msg.CommandArguments)
	case CommandNameMap[ResetMonthLimit].Command:
		answer, err = s.ResetUserLimit(ctx, msg.Message.UserID)
	default:
		answer = s.Help()
	}
	if err != nil {
		answer = err.Error()
	}

	duration := time.Since(startTime)
	metr.SummaryProcessTime.
		WithLabelValues(msg.CommandName).
		Observe(duration.Seconds())
	metr.HistogramProcessTime.
		WithLabelValues(msg.CommandName).
		Observe(duration.Seconds())

	return s.tgClient.SendMessage(answer, msg.Message.UserID)
}

func (s *Model) welcomeMessage() string {
	return "Welcomen!"
}

func (s *Model) addUser(ctx context.Context, userID int64) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_user_command")
	defer span.Finish()

	s.storage.AddUser(ctx, userID)
	return "", nil
}

func (s *Model) resetUser(ctx context.Context, userID int64) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "reset_user_command")
	defer span.Finish()

	if s.storage.ResetUser(ctx, userID) {
		return "Data erased!", nil
	}
	return "", errServer
}

// добавление категории
func (s *Model) AddCategory(ctx context.Context, userID int64, text string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_category_command")
	defer span.Finish()

	if text == "" {
		return "", errWrongCommandFormat
	}
	cat := strings.Split(text, " ")
	if len(cat) == 1 {
		if s.storage.AddCategory(ctx, userID, cat[0]) {
			return fmt.Sprintf("Category %s is added", cat[0]), nil
		}
		return fmt.Sprintf("Category %s is already added", cat[0]), nil
	}
	return "", errWrongCommandFormat
}

// вывод всех команд
func (s *Model) Help() string {
	var commandsSb strings.Builder
	keys := make([]int, 0, len(CommandNameMap))
	for k := range CommandNameMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		commandsSb.WriteString(
			fmt.Sprintf("/%s %s - %s.\n",
				CommandNameMap[k].Command,
				CommandNameMap[k].Format,
				CommandNameMap[k].Description))
	}
	return "Available commands:\n" + commandsSb.String()
}

// добавление траты
func (s *Model) AddExpence(ctx context.Context, userID int64, text string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_expence_command")
	defer span.Finish()

	if text == "" {
		return "", errWrongCommandFormat
	}
	commandArgs := strings.Split(text, " ")

	// проверка, что 3 аргумента
	if len(commandArgs) != 3 {
		return "", errWrongCommandFormat
	}

	// проверка, что 1ый аргумент (категория) добавлена
	if !s.storage.IsCategoryExists(ctx, userID, commandArgs[0]) {
		return "", errCategoryNotFound
	}

	// проверка, что 2ой аргумент (расход) является числом
	total, err := helpers.ConvertStringAmountToSub(commandArgs[1])
	if err != nil {
		return "", err
	}

	// проверка, что 3ий аргумент (дата) является датой
	date, err := helpers.StringToDate(commandArgs[2])
	if err != nil {
		return "", errDateWrongFormat
	}

	if err := s.storage.AddExpence(ctx, userID, commandArgs[0], total, date); err != nil {
		limitExceededError := &common.LimitExceededError{}
		if errors.As(err, &limitExceededError) {
			return "", err
		}
		return "", errCategoryNotFound
	}

	return "Expence added", nil
}

func (s *Model) GetReport(ctx context.Context, userID int64, text string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_report_command")
	defer span.Finish()

	commandArgs := strings.Split(text, " ")

	// проверка, что не больше 1 аргумента
	if len(commandArgs) > 1 {
		return "", errWrongCommandFormat
	}

	// таймстэмп, по которому будет сравниваться
	var limitTs time.Time = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Now().Location())

	switch commandArgs[0] {
	case "day":
		limitTs = helpers.GetStartOfCurrentDay()
	case "month":
		limitTs = helpers.GetStartOfCurrentMonth()
	case "year":
		limitTs = helpers.GetStartOfCurrentYear()
	default:
	}

	totalMap := s.storage.GetExpencesMap(ctx, userID, limitTs)

	if len(totalMap) == 0 {
		return "No expences!", nil
	}

	var rvSb strings.Builder
	if commandArgs[0] != "" {
		rvSb.WriteString(fmt.Sprintf("Last %s expences\n", commandArgs[0]))
	} else {
		rvSb.WriteString("All time expences\n")
	}

	for k, v := range totalMap {
		rvSb.WriteString(fmt.Sprintf("%s: %s\n", k, helpers.ConvertSubToAmount(v)))
	}

	return rvSb.String(), nil
}

func (s *Model) ChangeCurrency(ctx context.Context, userID int64, text string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "change_currency_command")
	defer span.Finish()

	commandArgs := strings.Split(text, " ")

	// проверка, что не больше 1 аргумента
	if len(commandArgs) > 1 {
		return "", errWrongCommandFormat
	}

	if !s.storage.ChangeCurrency(ctx, userID, commandArgs[0]) {
		return "Currency not found", nil
	}

	return "Currency successfully changed", nil
}

func (s *Model) SetUserLimit(ctx context.Context, userID int64, text string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "set_user_limit_command")
	defer span.Finish()

	commandArgs := strings.Split(text, " ")

	// проверка, что не больше 1 аргумента
	if len(commandArgs) > 1 {
		return "", errWrongCommandFormat
	}

	total, err := helpers.ConvertStringAmountToSub(commandArgs[0])
	if err != nil {
		return "", err
	}

	if total < 1 {
		return "", errLimitIsTooSmall
	}

	if err := s.storage.SetUserLimit(ctx, userID, total); err != nil {
		return "", err
	}

	return "Month limit updated!", nil
}

func (s *Model) ResetUserLimit(ctx context.Context, userID int64) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "reset_user_limit_command")
	defer span.Finish()

	if err := s.storage.ResetUserLimit(ctx, userID); err != nil {
		return "", errResetLimit
	}
	return "Month limit reseted", nil
}
