package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis(addr, pass string, db int) {
    RedisClient = redis.NewClient(&redis.Options{
        Addr:     addr,
        Password: pass,
        DB:       db,
    })
}

// SetIdempotency сохраняем idempotency key
func SetIdempotency(ctx context.Context, key string, messageID int64, ttl time.Duration) error {
    return RedisClient.Set(ctx, "idemp:"+key, messageID, ttl).Err()
}

// GetIdempotency проверка
func GetIdempotency(ctx context.Context, key string) (int64, bool, error) {
    val, err := RedisClient.Get(ctx, "idemp:"+key).Result()
    if err == redis.Nil {
        return 0, false, nil
    }
    if err != nil {
        return 0, false, err
    }
    var id int64
    fmt.Sscanf(val, "%d", &id)
    return id, true, nil
}

// SetLatestRoot
func SetLatestRoot(ctx context.Context, chatID int64, root []byte) error {
    return RedisClient.Set(ctx, fmt.Sprintf("chat:%d:latest_root", chatID), root, 0).Err()
}

// GetLatestRoot
func GetLatestRoot(ctx context.Context, chatID int64) ([]byte, error) {
    return RedisClient.Get(ctx, fmt.Sprintf("chat:%d:latest_root", chatID)).Bytes()
}
