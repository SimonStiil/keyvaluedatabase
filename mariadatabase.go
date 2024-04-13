package main

import (
	"fmt"
	"os"

	"database/sql"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

type MariaDatabase struct {
	Initialized  bool
	Connection   *sql.DB
	Config       *ConfigMysql
	Password     string
	DatabaseName string
}

type ConfigMysql struct {
	Address         string `mapstructure:"address"`
	Username        string `mapstructure:"username"`
	DatabaseName    string `mapstructure:"databaseName"`
	TableName       string `mapstructure:"tableName"`
	EnvVariableName string `mapstructure:"envVariableName"`
	KeyName         string `mapstructure:"keyName"`
	ValueName       string `mapstructure:"valueName"`
}

func MariaDBGetDefaults(configReader *viper.Viper) {
	configReader.SetDefault("address", "localhost:3306")
	configReader.SetDefault("username", "kvdb")
	configReader.SetDefault("databaseName", "")
	configReader.SetDefault("tableName", "kvdb")
	configReader.SetDefault("envVariableName", BaseENVname+"_MYSQL_PASSWORD")
	configReader.SetDefault("keyName", "key")
	configReader.SetDefault("valueName", "value")
}

func (MDB *MariaDatabase) Init() {

	logger.Debug("Initializing MariaDB", "function", "Init", "struct", "MariaDatabase")
	if MDB.Config.DatabaseName == "" {
		MDB.DatabaseName = MDB.Config.Username
	} else {
		MDB.DatabaseName = MDB.Config.DatabaseName
	}
	MDB.Password = os.Getenv(MDB.Config.EnvVariableName)
	var err error
	connectionString := fmt.Sprintf("%v:%v@tcp(%v)/%v", MDB.Config.Username, MDB.Password, MDB.Config.Address, MDB.DatabaseName)

	logger.Debug("Connection string: "+connectionString, "function", "Init", "struct", "MariaDatabase")
	MDB.Connection, err = sql.Open("mysql", connectionString)
	if err != nil {
		panic(err.Error())
	}
	MDB.Connection.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` ( `%v` CHAR(%v) PRIMARY KEY, `%v` VARCHAR(%v) NOT NULL) ENGINE = InnoDB; ", MDB.Config.TableName, MDB.Config.KeyName, rest.KeyMaxLength, MDB.Config.ValueName, rest.ValueMaxLength))

	logger.Debug("Initialization complete", "function", "Init", "struct", "MariaDatabase")
	MDB.Initialized = true
}

func (MDB *MariaDatabase) Set(key string, value interface{}) {
	if !MDB.Initialized {
		panic("F Unable to set. db not initialized()")
	}
	statement, err := MDB.Connection.Prepare(fmt.Sprintf("INSERT INTO `%v` (`%v`, `%v`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `%v`=?", MDB.Config.TableName, MDB.Config.KeyName, MDB.Config.ValueName, MDB.Config.ValueName))
	if err != nil {
		panic(err)
	}
	_, err = statement.Exec(key, value, value)
	if err != nil {
		panic(err)
	}
}

func (MDB *MariaDatabase) Get(key string) (string, bool) {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	rows, err := MDB.Connection.Query(fmt.Sprintf("select * from `%v` where `%v` = ? ", MDB.Config.TableName, MDB.Config.KeyName), key)

	if err != nil {
		logger.Error("Query failed with error", "function", "Get", "struct", "MariaDatabase", "error", err)
	}
	defer rows.Close()
	var kvpair rest.KVPairV1
	found := false
	for rows.Next() {
		err = rows.Scan(&kvpair.Key, &kvpair.Value)
		if err != nil {
			logger.Error("Scan row failed with error", "function", "Get", "struct", "MariaDatabase", "error", err)
		}
		found = true
	}
	return kvpair.Value, found
}

func (MDB *MariaDatabase) Keys() []string {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	rows, err := MDB.Connection.Query(fmt.Sprintf("select `%v` from `%v`", MDB.Config.KeyName, MDB.Config.TableName))
	if err != nil {
		logger.Error("Query failed with error", "function", "Keys", "struct", "MariaDatabase", "error", err)
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			logger.Error("Scan row failed with error", "function", "Keys", "struct", "MariaDatabase", "error", err)
		}
		keys = append(keys, key)
	}
	return keys
}

func (MDB *MariaDatabase) Delete(key string) {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	stmt, err := MDB.Connection.Prepare(fmt.Sprintf("delete from `%v` where `%v` = ?", MDB.Config.TableName, MDB.Config.KeyName))
	if err != nil {
		logger.Error("Prepare failed with error", "function", "Delete", "struct", "MariaDatabase", "error", err)
	}
	_, err = stmt.Exec(key)
	if err != nil {
		logger.Error("Exec failed with error", "function", "Delete", "struct", "MariaDatabase", "error", err)
	}
}

func (MDB *MariaDatabase) IsInitialized() bool {
	return MDB.Initialized
}

func (MDB *MariaDatabase) Close() {
	if !MDB.Initialized {
		panic("F Unable to close. db not initialized()")
	}
	err := MDB.Connection.Close()
	if err != nil {
		panic(err)
	}
	logger.Debug("Closed database connection", "function", "Close", "struct", "MariaDatabase")
}
