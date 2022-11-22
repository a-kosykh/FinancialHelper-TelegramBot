package database

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
)

type UsersDB struct {
	db *sql.DB
}

func NewUsersDB(db *sql.DB) *UsersDB {
	return &UsersDB{db}
}

func (db *UsersDB) IsUserAdded(ctx context.Context, user domain.User) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "is_user_added_db")
	defer span.Finish()

	var exists bool = false
	builder := sq.Select("1").From("users").Where(sq.Eq{
		"id": user.UserID,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return exists, err
	}

	err = db.db.QueryRowContext(ctx, query, args...).Scan(&exists)

	return exists, err
}

func (db *UsersDB) AddUser(ctx context.Context, user domain.User) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_user_db")
	defer span.Finish()

	builder := sq.Insert("users").Columns("id").Values(user.UserID)
	query, args, err := builder.PlaceholderFormat(sq.Dollar).ToSql()

	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(ctx, query, args...)

	return err
}

func (db *UsersDB) ResetUser(ctx context.Context, user domain.User) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "reset_user_db")
	defer span.Finish()

	builder := sq.Delete("users").Where(sq.Eq{
		"id": user.UserID,
	}).PlaceholderFormat(sq.Dollar)
	query, args, err := builder.ToSql()

	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(ctx, query, args...)

	return err
}

func (db *UsersDB) ChangeCurrency(ctx context.Context, user domain.User, currency domain.Currency) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "change_currency_db")
	defer span.Finish()

	builder := sq.Update("users").Set("base_currency_id", currency.ID).Where(sq.Eq{
		"id": user.UserID,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(ctx, query, args...)
	return err
}

func (db *UsersDB) GetUserBaseCurrency(ctx context.Context, user domain.User) (domain.Currency, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_user_base_currency_db")
	defer span.Finish()

	var rv domain.Currency = domain.Currency{
		ID: -1,
	}
	builder := sq.Select("base_currency_id").From("users").Where(sq.Eq{
		"id": user.UserID,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return rv, err
	}

	err = db.db.QueryRowContext(ctx, query, args...).Scan(&rv.ID)

	return rv, err
}

func (db *UsersDB) UpdateMonthLimits(ctx context.Context) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "update_month_limits_db")
	defer span.Finish()

	_, err := db.db.ExecContext(ctx, "UPDATE users SET current_month_limit = default_month_limit")
	return err
}

func (db *UsersDB) SetUserLimit(ctx context.Context, user domain.User) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "set_user_limit_db")
	defer span.Finish()

	builder := sq.Update("users").Set("default_month_limit", user.DefaultMonthLimit).Set("current_month_limit", user.CurrentMonthLimit).Where(sq.Eq{
		"id": user.UserID,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(ctx, query, args...)

	return err
}

func (db *UsersDB) ResetUserLimit(ctx context.Context, user domain.User) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "reset_user_limit_db")
	defer span.Finish()

	_, err := db.db.ExecContext(ctx, "UPDATE users SET current_month_limit = default_month_limit WHERE id=$1", user.UserID)
	return err
}
