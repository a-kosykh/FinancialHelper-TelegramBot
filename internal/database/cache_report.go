package database

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"gitlab.ozon.dev/akosykh114/telegram-bot/internal/domain"
)

type ReportCacheDb struct {
	rdb *redis.Client
}

func NewReportCacheDb(rdb *redis.Client) *ReportCacheDb {
	return &ReportCacheDb{rdb}
}

func (db *ReportCacheDb) GetUserExpences(ctx context.Context, user domain.User, limitTs time.Time) (map[string]int64, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "get_user_expences_cache")
	defer span.Finish()

	key := strconv.FormatInt(user.UserID, 10) + strconv.FormatInt(limitTs.Unix(), 10)
	cmd := db.rdb.HGetAll(ctx, key)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}

	cmdRv := cmd.Val()
	if len(cmdRv) <= 0 {
		return nil, nil
	}

	var rv map[string]int64 = make(map[string]int64)
	for k, v := range cmdRv {
		total, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		rv[k] = total
	}
	return rv, nil
}

func (db *ReportCacheDb) SetUserExpences(ctx context.Context, user domain.User, expencesMap map[string]string, limitTs time.Time) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "set_user_expences_cache")
	defer span.Finish()

	key := strconv.FormatInt(user.UserID, 10) + strconv.FormatInt(limitTs.Unix(), 10)
	cmd := db.rdb.HSet(ctx, key, expencesMap)
	if cmd.Err() != nil {
		return cmd.Err()
	}

	_, err := db.rdb.Expire(ctx, key, time.Minute*10).Result()
	if err != nil {
		return err
	}

	return nil
}

func (db *ReportCacheDb) DeleteUserReports(ctx context.Context, user domain.User) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "clear_report_cache")
	defer span.Finish()

	keyPattern := strconv.FormatInt(user.UserID, 10) + "*"

	keys, err := db.rdb.Keys(ctx, keyPattern).Result()
	if err != nil {
		return err
	}
	if len(keys) <= 0 {
		return nil
	}

	_, err = db.rdb.Del(ctx, keys...).Result()
	return err
}
