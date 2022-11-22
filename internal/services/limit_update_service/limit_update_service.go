package limitupdateservice

import (
	"context"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
)

type LimitUpdaterService struct {
	monthLimitChan chan struct{}
}

func New() (*LimitUpdaterService, chan struct{}) {
	rv := &LimitUpdaterService{
		monthLimitChan: make(chan struct{}),
	}
	return rv, rv.monthLimitChan
}

func formatServiceLog(log string) string {
	return fmt.Sprintf("<LimUpdServ>: %s", log)
}

func (s *LimitUpdaterService) StartService(ctx context.Context, wg *sync.WaitGroup) {
	logger.Info(formatServiceLog("Starting service..."))

	wg.Add(1)
	go func() {
		defer wg.Done()

		c := cron.New()
		if _, err := c.AddFunc("@monthly", func() {
			logger.Info(formatServiceLog("updating limits..."))
			s.monthLimitChan <- struct{}{}
		}); err != nil {
			logger.Error("cron func error", zap.Error(err))
			return
		}

		c.Start()

		<-ctx.Done()
		c.Stop()

		logger.Info(formatServiceLog("Stopping..."))
	}()
}
