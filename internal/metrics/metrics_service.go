package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
)

var (
	MessageReceived = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "tg_bot",
		Name:      "message_received_total",
		Help:      "The total number of processed messages",
	})

	SummaryProcessTime = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: "tg_bot",
		Name:      "summary_process_time_seconds",
		Objectives: map[float64]float64{
			0.5:  0.1,
			0.9:  0.01,
			0.99: 0.001,
		},
	}, []string{"command"})

	HistogramProcessTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tg_bot",
			Name:      "histogram_process_time_seconds",
			Buckets:   []float64{0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05},
		},
		[]string{"command"},
	)
)

func formatServiceMsg(log string) string {
	return "<Metric Service>: " + log
}

func StartService(ctx context.Context, wg *sync.WaitGroup) {
	logger.Info(formatServiceMsg("Starting service..."))

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := &http.Server{Addr: "localhost:8080", Handler: nil}

		go func() {
			if err := server.ListenAndServe(); err != nil {
				logger.Error("prometheus listen-serve error", zap.Error(err))
				return
			}
		}()

		http.Handle("/metrics", promhttp.Handler())

		<-ctx.Done()

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("prometheus shutdown error", zap.Error(err))
		}

		logger.Info(formatServiceMsg("Stopping..."))

	}()
}
