package database

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
)

type CategoriesDB struct {
	db *sql.DB
}

func NewCategoriesDB(db *sql.DB) *CategoriesDB {
	return &CategoriesDB{db}
}

func (db *CategoriesDB) IsCategoryExists(ctx context.Context, category domain.ExpenceCategory) (int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "is_cat_exists_db")
	defer span.Finish()

	var categoryID int64 = -1
	builder := sq.Select("id").From("expence_category").Where(sq.Eq{
		"user_id": category.UserID,
		"name":    category.Name,
	}).PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return categoryID, err
	}

	err = db.db.QueryRowContext(ctx, query, args...).Scan(&categoryID)
	if err != nil {
		return categoryID, err
	}

	return categoryID, nil
}

func (db *CategoriesDB) AddCategory(ctx context.Context, category domain.ExpenceCategory) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "add_category_db")
	defer span.Finish()

	builder := sq.Insert("expence_category").Columns("user_id", "name").Values(category.UserID, category.Name).PlaceholderFormat(sq.Dollar)
	query, args, err := builder.ToSql()

	if err != nil {
		return err
	}

	_, err = db.db.ExecContext(ctx, query, args...)

	return err
}
