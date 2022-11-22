package tg

import (
	"context"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/uber/jaeger-client-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	metrics "gitlab.ozon.dev/akosykh114/telegram-bot/internal/metrics"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/model/messages"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/tracing"
	"go.uber.org/zap"
)

type Client struct {
	client *tgbotapi.BotAPI
}

func New(token string) (*Client, error) {
	client, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "NewBotAPI")
	}

	return &Client{
		client: client,
	}, nil
}

func (c *Client) SendMessage(text string, userID int64) error {
	_, err := c.client.Send(tgbotapi.NewMessage(userID, text))
	if err != nil {
		return errors.Wrap(err, "client.Send")
	}
	return nil
}

func (c *Client) ListenUpdates(ctx context.Context, wg *sync.WaitGroup, msgModel *messages.Model) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Инициализация объекта трейсинга
		tracer, closer := tracing.Init("tg-bot")
		defer closer.Close()
		opentracing.SetGlobalTracer(tracer)

		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		updates := c.client.GetUpdatesChan(u)

		logger.Info("<Bot>: Listening to messages...")

		for {
			select {
			case update := <-updates:
				processMessage(ctx, update, msgModel)

			case <-ctx.Done():
				logger.Info("<Bot>: Stopping listening to messages...")
				return
			}
		}
	}()
}

func processMessage(ctx context.Context, update tgbotapi.Update, msgModel *messages.Model) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "process_message")
	defer span.Finish()

	if sc, ok := span.Context().(jaeger.SpanContext); ok {
		logger.Info("trace-info", zap.String("id", sc.TraceID().String()))
	}

	if update.Message == nil {
		return
	}

	metrics.MessageReceived.Inc()

	span.LogKV(
		"message", update.Message.Text,
		"userId", update.Message.From.UserName,
	)

	logger.Info(
		"message-info",
		zap.String("username", update.Message.From.UserName),
		zap.String("text", update.Message.Text),
	)
	msg := messages.Message{
		UserID: update.Message.From.ID,
	}
	var err error

	switch {
	case update.Message.IsCommand():
		func() {
			storageCtx, cancel := context.WithCancel(ctx)
			defer cancel()

			err = msgModel.IncomingCommandMessage(storageCtx, messages.CommandMessage{
				Message:          msg,
				CommandName:      update.Message.Command(),
				CommandArguments: update.Message.CommandArguments(),
			})
		}()
	default:
		err = msgModel.IncomingPlainTextMessage(messages.PlainTextMessage{
			Message: msg,
			Text:    update.Message.Text,
		})
	}

	if err != nil {
		logger.Error("error processing message:", zap.Error(err))
	}
}
