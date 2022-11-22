package storage

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/logger"
	"go.uber.org/zap"
)

type UsersDatabase interface {
	IsUserAdded(ctx context.Context, user domain.User) (bool, error)
	AddUser(ctx context.Context, user domain.User) error
	ResetUser(ctx context.Context, user domain.User) error
	ChangeCurrency(ctx context.Context, user domain.User, currency domain.Currency) error
	GetUserBaseCurrency(ctx context.Context, user domain.User) (domain.Currency, error)
	SetUserLimit(ctx context.Context, user domain.User) error
	UpdateMonthLimits(ctx context.Context) error
	ResetUserLimit(ctx context.Context, user domain.User) error
}

type CategoriesDatabase interface {
	IsCategoryExists(ctx context.Context, category domain.ExpenceCategory) (int64, error)
	AddCategory(ctx context.Context, category domain.ExpenceCategory) error
}

type CurrunciesDatabase interface {
	IsCurrencyExists(ctx context.Context, currency domain.Currency) (domain.Currency, error)
	GetCurrencyRate(ctx context.Context, currency domain.Currency) (domain.Currency, error)
	UpdateRates(ctx context.Context, currencies []domain.Currency) error
}

type ExpencesDatabase interface {
	AddExpence(ctx context.Context, expence domain.Expence) error
	GetUserExpences(ctx context.Context, user domain.User, limitTs time.Time) ([]domain.Expence, error)
}

type ReportRequester interface {
	GetReportRequestChan() chan domain.ReportRequest
}

type ExpencesGetter interface {
	GetReportExpencesChan() chan []domain.Expence
}

type ReportCacheDatabase interface {
	GetUserExpences(ctx context.Context, user domain.User, limitTs time.Time) (map[string]int64, error)
	SetUserExpences(ctx context.Context, user domain.User, expencesMap map[string]string, limitTs time.Time) error
	DeleteUserReports(ctx context.Context, user domain.User) error
}

type Storage struct {
	UsersDB           UsersDatabase
	CategoriesDB      CategoriesDatabase
	CurrunciesDB      CurrunciesDatabase
	ExpencesDB        ExpencesDatabase
	ReportCDB         ReportCacheDatabase
	ReportReq         ReportRequester
	ExpencesGetterObj ExpencesGetter
}

func New(
	usersDB UsersDatabase,
	categoriesDB CategoriesDatabase,
	currunciesDB CurrunciesDatabase,
	expencesDB ExpencesDatabase,
	reportCDB ReportCacheDatabase,
	reportRequester ReportRequester,
	expencesGetter ExpencesGetter,
) *Storage {
	return &Storage{
		UsersDB:           usersDB,
		CategoriesDB:      categoriesDB,
		CurrunciesDB:      currunciesDB,
		ExpencesDB:        expencesDB,
		ReportCDB:         reportCDB,
		ReportReq:         reportRequester,
		ExpencesGetterObj: expencesGetter,
	}
}

func (s *Storage) IsUserAdded(ctx context.Context, userID int64) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, "is_user_added_storage")
	defer span.Finish()

	found, _ := s.UsersDB.IsUserAdded(ctx, domain.User{UserID: userID})
	return found
}

func (s *Storage) AddUser(ctx context.Context, userID int64) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_user_storage")
	defer span.Finish()

	if err := s.UsersDB.AddUser(ctx, domain.User{UserID: userID, BaseCurrencyID: 0}); err != nil {
		logger.Warn("add user storage error:", zap.Error(err))
		return false
	}
	return true
}

func (s *Storage) ResetUser(ctx context.Context, userID int64) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, "reset_user_storage")
	defer span.Finish()

	if err := s.UsersDB.ResetUser(ctx, domain.User{UserID: userID}); err != nil {
		logger.Warn("reset user storage error:", zap.Error(err))
		return false
	}
	return true
}

func (s *Storage) ChangeCurrency(ctx context.Context, userID int64, currency string) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, "change_currency_storage")
	defer span.Finish()

	curr, err := s.CurrunciesDB.IsCurrencyExists(ctx, domain.Currency{Code: currency})
	if err != nil {
		return false
	}
	if err := s.UsersDB.ChangeCurrency(ctx, domain.User{UserID: userID}, curr); err != nil {
		logger.Warn("change currency storage error:", zap.Error(err))
		return false
	}

	err = s.ReportCDB.DeleteUserReports(ctx, domain.User{UserID: userID})
	if err != nil {
		logger.Warn("ChangeCurrency delete user reports error:", zap.Error(err))
	}

	return true
}

func (s *Storage) IsCategoryExists(ctx context.Context, userID int64, cat string) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, "is_category_exists_storage")
	defer span.Finish()

	idx, err := s.CategoriesDB.IsCategoryExists(ctx, domain.ExpenceCategory{
		UserID: userID,
		Name:   cat,
	})
	if err != nil {
		logger.Warn("is_category_exists storage error:", zap.Error(err))
		return false
	}
	return idx != -1
}

func (s *Storage) AddCategory(ctx context.Context, userID int64, cat string) bool {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_category_storage")
	defer span.Finish()

	if !s.IsCategoryExists(ctx, userID, cat) {
		err := s.CategoriesDB.AddCategory(ctx, domain.ExpenceCategory{UserID: userID, Name: cat})
		if err != nil {
			logger.Warn("add_category storage error:", zap.Error(err))
			return false
		}
		return true
	}
	return false
}

func (s *Storage) AddExpence(ctx context.Context, userID int64, cat string, total int64, date time.Time) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_expence_storage")
	defer span.Finish()

	categoryID, err := s.CategoriesDB.IsCategoryExists(ctx, domain.ExpenceCategory{
		UserID: userID,
		Name:   cat,
	})
	if err != nil {
		logger.Warn("AddExpence storage error:", zap.Error(err))
		return err
	}

	baseCurrency, err := s.UsersDB.GetUserBaseCurrency(ctx, domain.User{UserID: userID})
	if err != nil {
		logger.Warn("AddExpence storage error:", zap.Error(err))
		return err
	}

	baseCurrency, err = s.CurrunciesDB.GetCurrencyRate(ctx, domain.Currency{ID: baseCurrency.ID})
	if err != nil {
		logger.Warn("AddExpence storage error:", zap.Error(err))
		return err
	}

	expence := domain.Expence{
		UserID:     userID,
		CategoryID: categoryID,
		Timestamp:  date,
		Total:      int64(float64(total) / baseCurrency.Rate),
	}

	err = s.ExpencesDB.AddExpence(ctx, expence)
	if err != nil {
		logger.Warn("AddExpence storage error:", zap.Error(err))
		return err
	}

	err = s.ReportCDB.DeleteUserReports(ctx, domain.User{UserID: userID})
	if err != nil {
		logger.Warn("AddExpence storage error:", zap.Error(err))
	}

	return nil
}

func (s *Storage) GetExpencesMap(ctx context.Context, userID int64, limitTs time.Time) map[string]int64 {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_expence_map_storage")
	defer span.Finish()

	// обращение к кэшу за отчётом по пользователю
	rv, err := s.ReportCDB.GetUserExpences(ctx, domain.User{UserID: userID}, limitTs)
	if err != nil {
		logger.Warn("cache get report error", zap.Error(err))
	}
	// если произошёл cache-miss
	if rv == nil {
		baseCurrency, err := s.UsersDB.GetUserBaseCurrency(ctx, domain.User{UserID: userID})
		if err != nil {
			logger.Warn("GetExpencesMap storage error:", zap.Error(err))
			return nil
		}

		baseCurrency, err = s.CurrunciesDB.GetCurrencyRate(ctx, domain.Currency{ID: baseCurrency.ID})
		if err != nil {
			logger.Warn("GetExpencesMap storage error:", zap.Error(err))
			return nil
		}

		rv = make(map[string]int64, 0)
		rvForCache := make(map[string]string, 0)

		//expences, err := s.ExpencesDB.GetUserExpences(ctx, domain.User{UserID: userID}, limitTs)

		s.ReportReq.GetReportRequestChan() <- domain.ReportRequest{
			UserID:    userID,
			Timestamp: limitTs,
		}

		var expences []domain.Expence

		select {
		case <-ctx.Done():
			return nil
		case expences = <-s.ExpencesGetterObj.GetReportExpencesChan():
			//
		}

		if err != nil {
			logger.Warn("GetExpencesMap storage error:", zap.Error(err))
			return rv
		}
		for _, val := range expences {
			rv[val.CategoryName] += int64(float64(val.Total) * baseCurrency.Rate)
			rvForCache[val.CategoryName] = strconv.FormatInt(rv[val.CategoryName], 10)
		}
		logger.Info("report", zap.String("size", strconv.Itoa(len(rv))))
		err = s.ReportCDB.SetUserExpences(ctx, domain.User{UserID: userID}, rvForCache, limitTs)
		if err != nil {
			logger.Warn("SetExpencesMap storage error:", zap.Error(err))
		}
	}

	return rv
}

func (s *Storage) SetUserLimit(ctx context.Context, userID int64, total int64) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "set_user_limit_storage")
	defer span.Finish()

	baseCurrency, err := s.UsersDB.GetUserBaseCurrency(ctx, domain.User{UserID: userID})
	if err != nil {
		logger.Warn("SetUserLimit storage error:", zap.Error(err))
		return nil
	}

	baseCurrency, err = s.CurrunciesDB.GetCurrencyRate(ctx, domain.Currency{ID: baseCurrency.ID})
	if err != nil {
		logger.Warn("SetUserLimit storage error:", zap.Error(err))
		return nil
	}

	return s.UsersDB.SetUserLimit(ctx, domain.User{
		UserID:            userID,
		DefaultMonthLimit: int64(float64(total) / baseCurrency.Rate),
		CurrentMonthLimit: int64(float64(total) / baseCurrency.Rate),
	})
}

func (s *Storage) ResetUserLimit(ctx context.Context, userID int64) error {
	return s.UsersDB.ResetUserLimit(ctx, domain.User{UserID: userID})
}

func (s *Storage) WaitNewExchangeRates(ctx context.Context, wg *sync.WaitGroup, ch <-chan []domain.Currency) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case res := <-ch:
				func() {
					storageCtx, cancel := context.WithCancel(ctx)
					defer cancel()

					if err := s.CurrunciesDB.UpdateRates(storageCtx, res); err != nil {
						logger.Error("SetUserLimit storage error:", zap.Error(err))
					}
				}()
			case <-ctx.Done():
				logger.Info("Stopping listening to rate service...")
				return
			}
		}
	}()
}

func (s *Storage) WaitNewMonthLimit(ctx context.Context, wg *sync.WaitGroup, ch <-chan struct{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ch:
				func() {
					storageCtx, cancel := context.WithCancel(ctx)
					defer cancel()

					if err := s.UsersDB.UpdateMonthLimits(storageCtx); err != nil {
						logger.Error("Update month limits error:", zap.Error(err))
					}
				}()
			case <-ctx.Done():
				logger.Info("Stopping listening to limit updater service...")
				return
			}
		}
	}()
}
