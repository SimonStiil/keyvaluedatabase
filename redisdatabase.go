package main

import (
	"context"
	"fmt"
	"os"

	redis "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

type RedisDatabase struct {
	Initialized bool
	CTX         context.Context
	RDC         *redis.Client
	Config      *ConfigRedis
	Password    string
}

type ConfigRedis struct {
	Address         string `mapstructure:"address"`
	Prefix          string `mapstructure:"prefix"`
	SystemNS        string `mapstructure:"systemnamespace"`
	Seperator       string `mapstructure:"seperator"`
	EnvVariableName string `mapstructure:"envVariableName"`
}

func RedisDBGetDefaults(configReader *viper.Viper) {
	configReader.SetDefault("redis.address", "localhost:6379")
	configReader.SetDefault("redis.prefix", "kvdb")
	configReader.SetDefault("redis.systemnamespace", "kvdb")
	configReader.SetDefault("redis.seperator", "_")
	configReader.SetDefault("redis.envVariableName", BaseENVname+"_REDIS_PASSWORD")
}

func (DB *RedisDatabase) GetSystemNS() string {
	return DB.Config.SystemNS
}

func (DB *RedisDatabase) Init() {

	logger.Debug("Initializing Redis Connection", "function", "Init", "struct", "RedisDatabase")

	DB.Password = os.Getenv(DB.Config.EnvVariableName)
	DB.CTX = context.Background()
	DB.RDC = redis.NewClient(&redis.Options{
		Addr:     DB.Config.Address,
		Password: DB.Password, // no password set
		DB:       1,           // use default DB
	})
	logger.Debug("Initialization complete", "function", "Init", "struct", "RedisDatabase")
	DB.Initialized = true
}

func (DB *RedisDatabase) formatKey(namespace string, key string) string {
	return fmt.Sprintf("%v%v%v%v%v", DB.Config.Prefix, DB.Config.Seperator, namespace, DB.Config.Seperator, key)
}

func (DB *RedisDatabase) Set(namespace string, key string, value interface{}) error {
	if !DB.Initialized {
		panic("Unable to set. db not initialized()")
	}
	err := DB.RDC.Set(DB.CTX, DB.formatKey(namespace, key), value, 0).Err() //0 is ttl
	if err != nil {
		return err
	}
	return nil
}

func (DB *RedisDatabase) Get(namespace string, key string) (string, error) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	val, err := DB.RDC.Get(DB.CTX, DB.formatKey(namespace, key)).Result()
	if err == redis.Nil {
		return "", &ErrNotFound{Value: key}
	} else if err != nil {
		return "", err
	}
	return val, nil
}

func (DB *RedisDatabase) Keys(namespace string) ([]string, error) {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	var val []string
	var err error
	if namespace == "" {
		val, err = DB.RDC.Keys(DB.CTX, fmt.Sprintf("%v%v*", DB.Config.Prefix, DB.Config.Seperator)).Result()
	} else {
		val, err = DB.RDC.Keys(DB.CTX, fmt.Sprintf("%v%v%v%v*", DB.Config.Prefix, DB.Config.Seperator, namespace, DB.Config.Seperator)).Result()
	}

	logger.Debug("List", "function", "Keys", "struct", "RedisDatabase", "values", val, "error", err)
	if err == redis.Nil {
		return val, nil
	} else if err != nil {
		return val, err
	}
	//TODO: Remove prefix from keys
	//if namespace == "" {
	//TODO: Add filtering to only get namespace names not all full keys
	//}
	return val, nil
}

func (DB *RedisDatabase) CreateNamespace(namespace string) error {
	if !DB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	// Stub. Concept does not exist missing keys list implementation
	return nil
}

func (DB *RedisDatabase) DeleteNamespace(namespace string) error {
	if !DB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	// Stub. Concept does not exist missing keys list implementation
	return nil
}

func (DB *RedisDatabase) DeleteKey(namespace string, key string) error {
	if !DB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	err := DB.RDC.Del(DB.CTX, DB.formatKey(namespace, key)).Err()
	if err == redis.Nil {
		return nil
	} else if err != nil {
		return err
	}
	return nil
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
	logger.Debug("Closed Connection", "function", "Close", "struct", "RedisDatabase")
}
