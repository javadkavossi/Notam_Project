package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/pkg/logging"
)

var redisClient *redis.Client

// InitRedis مقداردهی Redis و اتصال را انجام می‌دهد و لاگ می‌زند
func InitRedis(cfg *config.Config, logger logging.Logger) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:               fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password:           cfg.Redis.Password,
		DB:                 0,
		DialTimeout:        cfg.Redis.DialTimeout * time.Second,
		ReadTimeout:        cfg.Redis.ReadTimeout * time.Second,
		WriteTimeout:       cfg.Redis.WriteTimeout * time.Second,
		PoolSize:           cfg.Redis.PoolSize,
		PoolTimeout:        cfg.Redis.PoolTimeout,
		IdleTimeout:        500 * time.Millisecond,
		IdleCheckFrequency: cfg.Redis.IdleCheckFrequency * time.Millisecond,
	})

	_, err := redisClient.Ping().Result()
	if err != nil {
		logger.Error(logging.Redis, logging.RedisInternal, "Failed to connect to Redis", map[logging.ExtraKey]interface{}{
			logging.ErrorMessage: err.Error(),
			"Addr":               cfg.Redis.Host + ":" + cfg.Redis.Port,
		})
		return err
	}

	logger.Info(logging.Redis, logging.RedisInternal, "Connected to Redis successfully", map[logging.ExtraKey]interface{}{
		"Addr": cfg.Redis.Host + ":" + cfg.Redis.Port,
	})
	return nil
}

// GetRedis بازگشت client Redis
func GetRedis() *redis.Client {
	return redisClient
}

// CloseRedis اتصال Redis را می‌بندد و لاگ می‌زند
func CloseRedis(logger logging.Logger) {
	if redisClient != nil {
		err := redisClient.Close()
		if err != nil {
			logger.Error(logging.Redis, logging.RedisInternal, "Failed to close Redis connection", map[logging.ExtraKey]interface{}{
				logging.ErrorMessage: err.Error(),
			})
		} else {
			logger.Info(logging.Redis, logging.RedisInternal, "Redis connection closed successfully", nil)
		}
	}
}

// Set داده‌ای را در Redis ذخیره می‌کند و لاگ می‌زند
func Set[T any](c *redis.Client, key string, value T, duration time.Duration, logger logging.Logger) error {
	v, err := json.Marshal(value)
	if err != nil {
		logger.Error(logging.Redis, logging.RedisInternal, "Failed to marshal value for Redis", map[logging.ExtraKey]interface{}{
			logging.ErrorMessage: err.Error(),
			"Key":                key,
		})
		return err
	}

	err = c.Set(key, v, duration).Err()
	if err != nil {
		logger.Error(logging.Redis, logging.RedisInternal, "Failed to set value in Redis", map[logging.ExtraKey]interface{}{
			logging.ErrorMessage: err.Error(),
			"Key":                key,
		})
		return err
	}

	logger.Info(logging.Redis, logging.RedisInternal, "Value set in Redis successfully", map[logging.ExtraKey]interface{}{
		"Key":      key,
		"Duration": duration.Seconds(),
	})
	return nil
}

// Get داده‌ای را از Redis بازیابی می‌کند و لاگ می‌زند
func Get[T any](c *redis.Client, key string, logger logging.Logger) (T, error) {
	var dest T = *new(T)

	v, err := c.Get(key).Result()
	if err != nil {
		logger.Error(logging.Redis, logging.RedisInternal, "Failed to get value from Redis", map[logging.ExtraKey]interface{}{
			logging.ErrorMessage: err.Error(),
			"Key":                key,
		})
		return dest, err
	}

	err = json.Unmarshal([]byte(v), &dest)
	if err != nil {
		logger.Error(logging.Redis, logging.RedisInternal, "Failed to unmarshal Redis value", map[logging.ExtraKey]interface{}{
			logging.ErrorMessage: err.Error(),
			"Key":                key,
		})
		return dest, err
	}

	logger.Info(logging.Redis, logging.RedisInternal, "Value retrieved from Redis successfully", map[logging.ExtraKey]interface{}{
		"Key": key,
	})
	return dest, nil
}
