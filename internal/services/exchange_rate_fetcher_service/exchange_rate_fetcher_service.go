package exchangeratefetcherservice

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
)

type ExchangeResponse struct {
	IsSuccess bool               `json:"success"`
	Rates     map[string]float64 `json:"rates"`
}

type ServiceConfigurer interface {
	CurrencyApiURL() string
	ExchangeServiceFetchInterval() time.Duration
	RequestTimeout() time.Duration
	AvailableCurrencies() []domain.Currency
	BaseCurrency() string
}

type ExchangeFetcherService struct {
	exchangeChan chan []domain.Currency

	client http.Client
	config ServiceConfigurer

	availableCurrencies []domain.Currency
}

func New(config ServiceConfigurer) (*ExchangeFetcherService, chan []domain.Currency) {
	rv := &ExchangeFetcherService{
		exchangeChan: make(chan []domain.Currency),
		config:       config,
		client:       http.Client{},
	}

	return rv, rv.exchangeChan
}

func formatServiceMsg(log string) string {
	return "<ERF>: " + log
}

func (s *ExchangeFetcherService) StartService(ctx context.Context, wg *sync.WaitGroup) {
	logger.Info(formatServiceMsg("Starting service..."))

	ticker := time.NewTicker(s.config.ExchangeServiceFetchInterval())

	request, err := http.NewRequest(http.MethodGet, s.config.CurrencyApiURL(), nil)
	if err != nil {
		logger.Error("request init error", zap.Error(err))
		return
	}

	s.availableCurrencies = s.config.AvailableCurrencies()

	q := request.URL.Query()
	q.Add("base", s.config.BaseCurrency())
	q.Add("symbols", getCurrencyCodesStr(s.availableCurrencies))

	request.URL.RawQuery = q.Encode()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			err := s.fetchData(ctx, request)
			if err != nil {
				logger.Error("fetching data error", zap.Error(err))
			} else {
				select {
				case s.exchangeChan <- s.availableCurrencies:
					logger.Info(formatServiceMsg("Rate data successfully fetched!"))
				case <-ctx.Done():
					ticker.Stop()
					logger.Info(formatServiceMsg("Stopping..."))
					return
				}
			}

			select {
			case <-ticker.C:
				continue
			case <-ctx.Done():
				logger.Info(formatServiceMsg("Stopping..."))
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *ExchangeFetcherService) fetchData(ctx context.Context, request *http.Request) error {
	reqCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout())
	defer cancel()

	request = request.WithContext(reqCtx)
	resp, err := s.client.Do(request)
	if err != nil {
		return err
	}

	exchangeResponse := ExchangeResponse{}
	err = json.NewDecoder(resp.Body).Decode(&exchangeResponse)
	if err != nil {
		return err
	}

	s.fillAvailableCurrenciesWithUpdatedRates(exchangeResponse.Rates)
	return nil
}

func (s *ExchangeFetcherService) fillAvailableCurrenciesWithUpdatedRates(rates map[string]float64) {
	for i, c := range s.availableCurrencies {
		if _, found := rates[c.Code]; found {
			c.Rate = rates[c.Code]
			s.availableCurrencies[i] = c
		}
	}
}

func getCurrencyCodesStr(currencies []domain.Currency) string {
	keys := make([]string, len(currencies))

	for _, v := range currencies {
		keys = append(keys, v.Code)
	}
	return strings.Join(keys, ",")
}
