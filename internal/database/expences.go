package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/common"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/helpers"
)

type ExpencesDB struct {
	db *sql.DB
}

func NewExpencesDB(db *sql.DB) *ExpencesDB {
	return &ExpencesDB{db}
}

func (db *ExpencesDB) AddExpence(ctx context.Context, expence domain.Expence) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_expence_db")
	defer span.Finish()

	tx, err := db.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:all

	if expence.Timestamp.After(helpers.GetStartOfCurrentMonth()) {
		var monthLimit int64
		if err := tx.QueryRowContext(ctx, "UPDATE users SET current_month_limit = current_month_limit - $1 WHERE id = $2 RETURNING current_month_limit;", expence.Total, expence.UserID).Scan(&monthLimit); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("user not found")
			}
		}
		if monthLimit < 0 {
			return fmt.Errorf("add expence: %w", &common.LimitExceededError{})
		}
	}

	builder := sq.Insert("expences").Columns(
		"user_id",
		"category_id",
		"ts",
		"total",
	).Values(
		expence.UserID,
		expence.CategoryID,
		expence.Timestamp,
		expence.Total,
	).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return err
}

func (db *ExpencesDB) GetUserExpences(ctx context.Context, user domain.User, limitTs time.Time) ([]domain.Expence, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_user_expences_db")
	defer span.Finish()

	var expences []domain.Expence = nil

	builder := sq.Select("expence_category.name, expences.total").From("expences").Where(
		fmt.Sprintf("expences.user_id = %d AND expences.ts >= %s", user.UserID, limitTs.Format(`'2006-01-02 15:04:05'`)),
	).Join("expence_category ON expences.category_id = expence_category.id").PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()

	if err != nil {
		return expences, err
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return expences, err
	}

	expences = make([]domain.Expence, 0)
	for rows.Next() {
		var expence domain.Expence
		if err := rows.Scan(
			&expence.CategoryName,
			&expence.Total,
		); err != nil {
			return expences, err
		}
		expences = append(expences, expence)
	}

	if err = rows.Err(); err != nil {
		return expences, err
	}

	return expences, nil
}
