package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-redis/redis/v8"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/clients/tg"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/config"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/database"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	metrics "gitlab.ozon.dev/akosykh114/telegram-bot/internal/metrics"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/model/messages"
	exchangeratefetcherservice "gitlab.ozon.dev/akosykh114/telegram-bot/internal/services/exchange_rate_fetcher_service"
	grpcserver "gitlab.ozon.dev/akosykh114/telegram-bot/internal/services/grpc_server"
	limitupdateservice "gitlab.ozon.dev/akosykh114/telegram-bot/internal/services/limit_update_service"
	reportrequestproducer "gitlab.ozon.dev/akosykh114/telegram-bot/internal/services/report_request_producer"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/storage"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Инициализация логгера
	logger := logger.InitLogger("data/zap_config.json")

	// Запуск сервиса сбор метрик
	metrics.StartService(ctx, &wg)

	config, err := config.New()
	if err != nil {
		logger.Fatal("config init failed", zap.Error(err))
	}

	tgClient, err := tg.New(config.Token())
	if err != nil {
		logger.Fatal("tg client init failed", zap.Error(err))
	}

	// Запуск сервиса фетчинга актуального курса валют
	exchangeFetcherService, exchangeChan := exchangeratefetcherservice.New(config)
	exchangeFetcherService.StartService(ctx, &wg)

	// Запуск сервиса периодического обновления лимитов
	limitService, monthLimitChan := limitupdateservice.New()
	limitService.StartService(ctx, &wg)

	// Запуск gRPC сервера
	grpcServer := grpcserver.New()
	err = grpcServer.StartService(ctx, &wg)
	if err != nil {
		logger.Fatal("grpc-server init failed", zap.Error(err))
	}

	// Запуск продюссера реквеста отчётов
	reportRequestProducer, err := reportrequestproducer.New()
	if err != nil {
		logger.Fatal("report producer init error:", zap.Error(err))
	}
	reportRequestProducer.StartService(ctx, &wg)

	// Инициализация объектов слоя БД
	db, err := sql.Open("postgres", "host=localhost port=5432 dbname=telegram-bot-db user=postgres password=admin sslmode=disable")
	if err != nil {
		logger.Fatal("db open error", zap.Error(err))
	}
	usersDB := database.NewUsersDB(db)
	categoriesDB := database.NewCategoriesDB(db)
	currenciesDB := database.NewCurrenciesDB(db)
	expencesDB := database.NewExpencesDB(db)

	// Инициализация клиента Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	reportDB := database.NewReportCacheDb(rdb)

	// Инициализация хранилища
	storageModel := storage.New(usersDB, categoriesDB, currenciesDB, expencesDB, reportDB, reportRequestProducer, grpcServer)
	storageModel.WaitNewExchangeRates(ctx, &wg, exchangeChan)
	storageModel.WaitNewMonthLimit(ctx, &wg, monthLimitChan)

	// Запуск бота
	msgModel := messages.New(tgClient, storageModel)
	tgClient.ListenUpdates(ctx, &wg, msgModel)

	wg.Wait()

	if err := db.Close(); err != nil {
		logger.Fatal("db close error", zap.Error(err))
	}
}
