package main

import (
	"errors"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type ConfigUser struct {
	Username    string            `yaml:"username,omitempty"`
	Password    string            `yaml:"password,omitempty"`
	Permissions ConfigPermissions `yaml:"permissions,inline"`
}

type ConfigHosts struct {
	Address     string            `yaml:"address,omitempty"`
	Permissions ConfigPermissions `yaml:"permissions,inline"`
}

type ConfigRedis struct {
	Address      string `yaml:"address,omitempty"`
	VariableName string `yaml:"envVariableName,omitempty"`
	Password     string `yaml:"-"`
}
type ConfigMysql struct {
	Address      string `yaml:"address,omitempty"`
	Username     string `yaml:"username,omitempty"`
	DatabaseName string `yaml:"databaseName,omitempty"`
	TableName    string `yaml:"tableName,omitempty"`
	VariableName string `yaml:"envVariableName,omitempty"`
	Password     string `yaml:"-"`
}
type ConfigPrometheus struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Endpoint string `yaml:"endpoint,omitempty"`
}

type ConfigPermissions struct {
	Read  bool `yaml:"read,omitempty"`
	Write bool `yaml:"write,omitempty"`
	List  bool `yaml:"list,omitempty"`
}

type Config struct {
	Users          []ConfigUser     `yaml:"users,omitempty"`
	Hosts          []ConfigHosts    `yaml:"hosts,omitempty"`
	TrustedProxies []string         `yaml:"trustedProxies,omitempty"`
	Redis          ConfigRedis      `yaml:"redis,omitempty"`
	Mysql          ConfigMysql      `yaml:"mysql,omitempty"`
	Prometheus     ConfigPrometheus `yaml:"prometheus,omitempty"`
}

func ConfigRead(Filename string) Config {
	yamlFile, err := os.ReadFile(Filename)
	var document Config
	if err != nil {
		// https://stackoverflow.com/questions/12518876/how-to-check-if-a-file-exists-in-go
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	} else {
		err = yaml.Unmarshal(yamlFile, &document)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)

		} else {
			if debug {
				log.Println("ReadConfig - ", document)
			}
		}
	}
	//Read User from ENV
	username := os.Getenv(BaseENVname + "_AUTH_USERNAME")
	/*
		if username == "" && len(document.Users) == 0 {
			log.Fatalf("%v or KVDB_AUTH_USERNAME must be provided", Filename)
		}
	*/
	password := os.Getenv(BaseENVname + "_AUTH_PASSWORD")
	/*
		if password == "" && len(document.Users) == 0 {
			log.Fatalf("%v or KVDB_AUTH_PASSWORD must be provided", Filename)
		}
	*/
	if username != "" && password != "" {
		document.Users = append(document.Users, ConfigUser{
			Username: username,
			Password: AuthEncode(AuthHash(password)),
			Permissions: ConfigPermissions{
				Read:  true,
				Write: true,
				List:  true,
			}})
	}

	// Read Redis Config
	redisHostENV := os.Getenv(BaseENVname + "_REDIS_HOST")
	if redisHostENV != "" {
		document.Redis.Address = redisHostENV
	}
	if document.Redis.Address != "" {
		if document.Redis.VariableName != "" {
			document.Redis.Password = os.Getenv(document.Redis.VariableName)
		} else {
			document.Redis.Password = os.Getenv(BaseENVname + "_REDIS_PASSWORD")
		}
	}
	if document.Mysql.Address != "" {
		if document.Mysql.VariableName != "" {
			document.Mysql.Password = os.Getenv(document.Mysql.VariableName)
		} else {
			document.Mysql.Password = os.Getenv(BaseENVname + "_MYSQL_PASSWORD")
		}
	}
	return document
}
