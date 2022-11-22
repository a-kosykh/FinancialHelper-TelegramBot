package config

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gopkg.in/yaml.v3"
)

const configFile = "data/config.yaml"

type Config struct {
	Token                        string   `yaml:"token"`
	CurrencyApiURL               string   `yaml:"currency_api_url"`
	ExchangeServiceFetchInterval int      `yaml:"exchange_service_fetch_interval"`
	RequestTimeout               int      `yaml:"request_timeout"`
	BaseCurrency                 string   `yaml:"base_currency"`
	AvailableCurrencies          []string `yaml:"currencies"`
}

type Service struct {
	config Config
}

func New() (*Service, error) {
	s := &Service{
		config: Config{},
	}

	rawYAML, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading config file")
	}

	err = yaml.Unmarshal(rawYAML, &s.config)
	if err != nil {
		return nil, errors.Wrap(err, "parsing yaml")
	}

	return s, nil
}

func (s *Service) Token() string {
	return s.config.Token
}

func (s *Service) CurrencyApiURL() string {
	return s.config.CurrencyApiURL
}

func (s *Service) ExchangeServiceFetchInterval() time.Duration {
	return time.Duration(s.config.ExchangeServiceFetchInterval) * time.Second
}

func (s *Service) RequestTimeout() time.Duration {
	return time.Duration(s.config.RequestTimeout) * time.Second
}

func (s *Service) AvailableCurrencies() []domain.Currency {
	rv := make([]domain.Currency, 0)
	for i, v := range s.config.AvailableCurrencies {
		rv = append(rv, domain.Currency{
			ID:   i,
			Code: v,
			Rate: 1,
		})
	}
	return rv
}

func (s *Service) BaseCurrency() string {
	return s.config.BaseCurrency
}
