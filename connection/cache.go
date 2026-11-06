package connection

import (
	"chat/globals"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

var Cache *redis.Client

// redisWarned ensures the unreachable-redis warning is only logged once
var redisWarned = false

// redisAlive reflects the last health state of the redis connection;
// checked by middlewares to degrade gracefully without a redis instance
var redisAlive atomic.Bool

// RedisAlive reports whether the last connection attempt to redis succeeded
func RedisAlive() bool {
	return redisAlive.Load()
}

func InitRedisSafe() *redis.Client {
	ConnectRedis()

	// using Cache as a global variable to point to the latest redis connection
	RedisWorker(Cache)
	return Cache
}

func ConnectRedis() *redis.Client {
	// connect to redis
	Cache = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", viper.GetString("redis.host"), viper.GetInt("redis.port")),
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})

	if err := pingRedis(Cache); err != nil {
		redisAlive.Store(false)
		if !redisWarned {
			globals.Warn(
				fmt.Sprintf(
					"[connection] failed to connect to redis host: %s (message: %s), will retry in 5 seconds",
					viper.GetString("redis.host"),
					err.Error(),
				),
			)
			redisWarned = true
		}
	} else {
		redisAlive.Store(true)
		redisWarned = false
		globals.Debug(fmt.Sprintf("[connection] connected to redis (host: %s)", viper.GetString("redis.host")))
	}

	if viper.GetBool("debug") {
		Cache.FlushAll(context.Background())
		globals.Debug(fmt.Sprintf("[connection] flush redis cache (host: %s)", viper.GetString("redis.host")))
	}
	return Cache
}
