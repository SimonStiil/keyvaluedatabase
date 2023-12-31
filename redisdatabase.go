package main

import (
	"context"
	"log"
	"os"

	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type RedisDatabase struct {
	Initialized bool
	CTX         context.Context
	RDC         *redis.Client
	Config      *ConfigType
	Password    string
}

type ConfigRedis struct {
	Address         string `mapstructure:"address"`
	EnvVariableName string `mapstructure:"envVariableName"`
}

func RedisDBGetDefaults(configReader *viper.Viper) {
	configReader.SetDefault("redis.address", "localhost:6379")
	configReader.SetDefault("redis.envVariableName", BaseENVname+"_REDIS_PASSWORD")
}

func (DB *RedisDatabase) Init() {
	if DB.Config.Debug {
		log.Println("db.init (redis)")
	}

	DB.Password = os.Getenv(DB.Config.Redis.EnvVariableName)
	DB.CTX = context.Background()
	DB.RDC = redis.NewClient(&redis.Options{
		Addr:     DB.Config.Redis.Address,
		Password: DB.Password, // no password set
		DB:       1,           // use default DB
	})
	if DB.Config.Debug {
		log.Println("db.init - complete")
	}
	DB.Initialized = true
}

func (DB *RedisDatabase) Set(key string, value interface{}) {
	if !DB.Initialized {
		panic("Unable to set. db not initialized()")
	}
	err := DB.RDC.Set(DB.CTX, key, value, 0).Err() //0 is ttl
	if err != nil {
		panic(err)
	}
}

func (DB *RedisDatabase) Get(key string) (string, bool) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	val, err := DB.RDC.Get(DB.CTX, key).Result()
	if err == redis.Nil {
		return "", false
	} else if err != nil {
		panic(err)
	}
	return val, true
}

func (DB *RedisDatabase) Keys() []string {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	val, err := DB.RDC.Keys(DB.CTX, "*").Result()

	if DB.Config.Debug {
		log.Printf("DB.List: %v %v\n", val, err)
	}
	if err == redis.Nil {
		return val
	} else if err != nil {
		panic(err)
	}
	return val
}

func (DB *RedisDatabase) Delete(key string) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	err := DB.RDC.Del(DB.CTX, key).Err()
	if err == redis.Nil {
		return
	} else if err != nil {
		panic(err)
	}
}

func (DB *RedisDatabase) IsInitialized() bool {
	return DB.Initialized
}

func (DB *RedisDatabase) Close() {
	if !DB.Initialized {
		panic("Unable to close. db not initialized()")
	}
	err := DB.RDC.Close()
	if err != nil {
		panic(err)
	}
	if DB.Config.Debug {
		log.Println("db.closed")
	}
}
