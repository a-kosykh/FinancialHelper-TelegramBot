package messages

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redismock/v8"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/database"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/helpers"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	mocks "gitlab.ozon.dev/akosykh114/telegram-bot/internal/mocks/messages"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/storage"
)

type ReportRequestProducer struct{}

func (r *ReportRequestProducer) GetReportRequestChan() chan domain.ReportRequest {
	return make(chan domain.ReportRequest)
}

type ExpencesGetter struct{}

func (e *ExpencesGetter) GetReportExpencesChan() chan []domain.Expence {
	return make(chan []domain.Expence)
}

func Test_OnStartCommand_ShouldAnswerWithIntroMessage(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	rdb, _ := redismock.NewClientMock()
	reportDB := database.NewReportCacheDb(rdb)

	_ = logger.InitLogger("data/zap_config.json")

	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)

	r := &ReportRequestProducer{}
	e := &ExpencesGetter{}

	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, r, e)
	model := New(sender, storageModel)

	sender.EXPECT().SendMessage("Welcomen!", int64(123))

	ctx := context.Background()
	err = model.IncomingCommandMessage(ctx, CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "start",
		CommandArguments: "",
	})

	assert.NoError(t, err)
}

func Test_OnUnknownCommand_ShouldAnswerWithHelpMessage(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	_ = logger.InitLogger("data/zap_config.json")

	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	rdb, _ := redismock.NewClientMock()
	reportDB := database.NewReportCacheDb(rdb)

	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)

	r := &ReportRequestProducer{}
	e := &ExpencesGetter{}

	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, r, e)
	model := New(sender, storageModel)
	sender.EXPECT().SendMessage(model.Help(), int64(123))

	err = model.IncomingPlainTextMessage(PlainTextMessage{
		Message: Message{
			UserID: 123,
		},
		Text: "some text",
	})

	assert.NoError(t, err)
}

func Test_OnAddCategoryCommand_ShouldAnswerWithAddedMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	_ = logger.InitLogger("data/zap_config.json")

	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	rdb, _ := redismock.NewClientMock()
	reportDB := database.NewReportCacheDb(rdb)

	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)

	r := &ReportRequestProducer{}
	e := &ExpencesGetter{}

	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, r, e)
	model := New(sender, storageModel)

	columns := []string{"id"}
	mock.ExpectExec("INSERT INTO users").WithArgs(123).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns))
	mock.ExpectExec("INSERT INTO expence_category").WithArgs(123, "food").WillReturnResult(sqlmock.NewResult(1, 1))

	sender.EXPECT().SendMessage("Category food is added", int64(123))

	err = model.IncomingCommandMessage(context.Background(), CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "add_category",
		CommandArguments: "food",
	})
	assert.NoError(t, err)
}

func Test_OnAddCategoryCommand_ShouldAnswerWithErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	_ = logger.InitLogger("data/zap_config.json")

	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	rdb, _ := redismock.NewClientMock()
	reportDB := database.NewReportCacheDb(rdb)

	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	r := &ReportRequestProducer{}
	e := &ExpencesGetter{}

	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, r, e)
	model := New(sender, storageModel)

	columns := []string{"id"}
	mock.ExpectExec("INSERT INTO users").WithArgs(123).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns))
	mock.ExpectExec("INSERT INTO expence_category").WithArgs(123, "food").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO users").WithArgs(123).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns).AddRow(1))

	sender.EXPECT().SendMessage("Category food is added", int64(123))
	sender.EXPECT().SendMessage("Category food is already added", int64(123))

	_ = model.IncomingCommandMessage(context.Background(), CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "add_category",
		CommandArguments: "food",
	})
	assert.NoError(t, err)

	err = model.IncomingCommandMessage(context.Background(), CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "add_category",
		CommandArguments: "food",
	})
	assert.NoError(t, err)
}

func Test_OnAddExpenceCommand_ShouldAnswerWithNoCatErr(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	_ = logger.InitLogger("data/zap_config.json")

	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	rdb, _ := redismock.NewClientMock()
	reportDB := database.NewReportCacheDb(rdb)

	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	r := &ReportRequestProducer{}
	e := &ExpencesGetter{}

	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, r, e)
	model := New(sender, storageModel)

	columns := []string{"id"}
	mock.ExpectExec("INSERT INTO users").WithArgs(123).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns))

	sender.EXPECT().SendMessage(
		fmt.Sprintf("category was not found - use '/%s %s'",
			CommandNameMap[AddCategoryCmd].Command,
			CommandNameMap[AddCategoryCmd].Format),
		int64(123))

	err = model.IncomingCommandMessage(context.Background(), CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "add_expence",
		CommandArguments: "food 100 09/10/12",
	})
	assert.NoError(t, err)
}

func Test_OnAddExpenceCommand_ShouldAnswerWithExpAdded(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	_ = logger.InitLogger("data/zap_config.json")

	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	rdb, mocksRedis := redismock.NewClientMock()
	reportDB := database.NewReportCacheDb(rdb)

	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	r := &ReportRequestProducer{}
	e := &ExpencesGetter{}

	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, r, e)
	model := New(sender, storageModel)

	mocksRedis.ExpectKeys("123*")

	columns := []string{"id"}
	mock.ExpectExec("INSERT INTO users").WithArgs(123).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns))
	mock.ExpectExec("INSERT INTO expence_category").WithArgs(123, "food").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("INSERT INTO users").WithArgs(123).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns).AddRow(1))
	mock.ExpectQuery("SELECT id FROM expence_category").WithArgs("food", 123).WillReturnRows(mock.NewRows(columns).AddRow(1))
	mock.ExpectQuery("SELECT base_currency_id").WithArgs(123).WillReturnRows(mock.NewRows(columns).AddRow(1))
	mock.ExpectQuery("SELECT rate").WithArgs(1).WillReturnRows(mock.NewRows(columns).AddRow(1))

	mock.ExpectBegin()
	date, _ := helpers.StringToDate("09/10/2012")
	mock.ExpectExec("INSERT INTO expences").WithArgs(123, 1, date, 10000).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	sender.EXPECT().SendMessage("Category food is added", int64(123))
	sender.EXPECT().SendMessage("Expence added", int64(123))

	err = model.IncomingCommandMessage(context.Background(), CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "add_category",
		CommandArguments: "food",
	})
	assert.NoError(t, err)

	err = model.IncomingCommandMessage(context.Background(), CommandMessage{
		Message: Message{
			UserID: 123,
		},
		CommandName:      "add_expence",
		CommandArguments: "food 100 09/10/2012",
	})
	assert.NoError(t, err)
}
