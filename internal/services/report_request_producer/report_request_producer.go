package reportrequestproducer

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/Shopify/sarama"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
)

var (
	KafkaTopic = "report-topic"
	BrokerList = []string{"localhost:9092"}
)

type ReportRequestProducer struct {
	reportChan chan domain.ReportRequest
	producer   sarama.AsyncProducer
}

func New() (*ReportRequestProducer, error) {
	rv := &ReportRequestProducer{
		reportChan: make(chan domain.ReportRequest),
	}

	config := sarama.NewConfig()
	config.Version = sarama.V2_6_0_0
	config.Producer.Return.Successes = true

	p, err := sarama.NewAsyncProducer(BrokerList, config)
	if err != nil {
		return nil, err
	}

	go func() {
		for err := range p.Errors() {
			logger.Warn("Failed to write message:", zap.Error(err))
		}
	}()

	rv.producer = p

	return rv, nil
}

func (r *ReportRequestProducer) GetReportRequestChan() chan domain.ReportRequest {
	return r.reportChan
}

func formatServiceLog(log string) string {
	return "<Report Request Producer>: " + log
}

func (r *ReportRequestProducer) StartService(ctx context.Context, wg *sync.WaitGroup) {
	logger.Info(formatServiceLog("Starting producer..."))

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case report := <-r.reportChan:
				msg := sarama.ProducerMessage{
					Topic: KafkaTopic,
					Key:   sarama.StringEncoder(strconv.FormatInt(report.UserID, 10)),
					Value: sarama.StringEncoder(strconv.FormatInt(report.Timestamp.Unix(), 10)),
				}
				r.producer.Input() <- &msg
				successMsg := <-r.producer.Successes()

				logger.Info(formatServiceLog(fmt.Sprintf("Successfully written to topic, offset: %d", successMsg.Offset)))
			case <-ctx.Done():
				logger.Info(formatServiceLog("Stopping..."))
				err := r.producer.Close()
				if err != nil {
					logger.Fatal(formatServiceLog("Error closing producer:"), zap.Error(err))
				}
				return
			}
		}
	}()
}
