package database

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
)

type CurreciesDB struct {
	db *sql.DB
}

func NewCurrenciesDB(db *sql.DB) *CurreciesDB {
	return &CurreciesDB{db}
}

func (db *CurreciesDB) IsCurrencyExists(ctx context.Context, currency domain.Currency) (domain.Currency, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "is_currency_exists_db")
	defer span.Finish()

	builder := sq.Select("id").From("currency").Where(sq.Eq{
		"code": currency.Code,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		currency.ID = -1
		return currency, err
	}

	err = db.db.QueryRowContext(ctx, query, args...).Scan(&currency.ID)

	return currency, err
}

func (db *CurreciesDB) GetCurrencyRate(ctx context.Context, currency domain.Currency) (domain.Currency, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_currency_rate_db")
	defer span.Finish()

	builder := sq.Select("rate").From("currency").Where(sq.Eq{
		"id": currency.ID,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return currency, err
	}

	err = db.db.QueryRowContext(ctx, query, args...).Scan(&currency.Rate)

	return currency, err
}

func (db *CurreciesDB) UpdateRates(ctx context.Context, currencies []domain.Currency) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "update_rates_db")
	defer span.Finish()

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	builder := psql.Insert("currency").Columns("id", "code", "rate").PlaceholderFormat(sq.Dollar)
	for _, v := range currencies {
		builder = builder.Values(v.ID, v.Code, v.Rate)
	}
	builder = builder.Suffix("ON CONFLICT (id) DO UPDATE SET rate = EXCLUDED.rate")

	totalStr, vals, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(ctx, totalStr, vals...)

	return err
}
