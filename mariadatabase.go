package main

import (
	"fmt"
	"os"
	"strings"

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
	SystemTableName string `mapstructure:"systemTableName"`
	EnvVariableName string `mapstructure:"envVariableName"`
	KeyName         string `mapstructure:"keyName"`
	ValueName       string `mapstructure:"valueName"`
}

func MariaDBGetDefaults(configReader *viper.Viper) {
	configReader.SetDefault("mysql.address", "localhost:3306")
	configReader.SetDefault("mysql.username", "kvdb")
	configReader.SetDefault("mysql.databaseName", "")
	configReader.SetDefault("mysql.systemTableName", "kvdb")
	configReader.SetDefault("mysql.envVariableName", BaseENVname+"_MYSQL_PASSWORD")
	configReader.SetDefault("mysql.keyName", "key")
	configReader.SetDefault("mysql.valueName", "value")
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
	_, err = MDB.Connection.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` ( `%v` CHAR(%v) PRIMARY KEY, `%v` VARCHAR(%v) NOT NULL) ENGINE = InnoDB; ", MDB.Config.SystemTableName, MDB.Config.KeyName, rest.KeyMaxLength, MDB.Config.ValueName, rest.ValueMaxLength))
	if err != nil {
		panic(err.Error())
	}
	MDB.Initialized = true
	err = MDB.CreateNamespace(MDB.GetSystemNS())
	if err != nil {
		panic(err.Error())
	}
	logger.Debug("Initialization complete", "function", "Init", "struct", "MariaDatabase")
}
func (MDB *MariaDatabase) GetSystemNS() string {
	return MDB.Config.SystemTableName
}

func (MDB *MariaDatabase) Set(namespace string, key string, value interface{}) error {
	if !MDB.Initialized {
		panic("F Unable to set. db not initialized()")
	}
	MDB.CreateNamespace(namespace)
	statement, err := MDB.Connection.Prepare(fmt.Sprintf("INSERT INTO `%v` (`%v`, `%v`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `%v`=?", namespace, MDB.Config.KeyName, MDB.Config.ValueName, MDB.Config.ValueName))
	if err != nil {
		return err
	}
	_, err = statement.Exec(key, value, value)
	if err != nil {
		return err
	}
	return nil
}

func (MDB *MariaDatabase) Get(namespace string, key string) (string, error) {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	rows, err := MDB.Connection.Query(fmt.Sprintf("select * from `%v` where `%v` = ? ", namespace, MDB.Config.KeyName), key)

	if err != nil {
		if strings.Contains(err.Error(), "Error 1146 (42S02)") {
			return "", &ErrNotFound{Value: namespace}
		}
		logger.Error("Query failed with error", "function", "Get", "struct", "MariaDatabase", "namespace", namespace, "error", err)
		return "", err
	}
	defer rows.Close()
	kvpair := rest.KVPairV2{}
	found := false
	for rows.Next() {
		err = rows.Scan(&kvpair.Key, &kvpair.Value)
		if err != nil {
			logger.Error("Scan row failed with error", "function", "Get", "struct", "MariaDatabase", "namespace", namespace, "error", err)
			return "", err
		}
		found = true
	}
	if found {
		return kvpair.Value, err
	} else {
		return "", &ErrNotFound{Value: key}
	}
}

func (MDB *MariaDatabase) Keys(namespace string) ([]string, error) {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	var rows *sql.Rows
	var err error
	if namespace == "" {
		rows, err = MDB.Connection.Query("show TABLES")
	} else {
		rows, err = MDB.Connection.Query(fmt.Sprintf("select `%v` from `%v`", MDB.Config.KeyName, namespace))
	}
	if err != nil {
		logger.Error("Query failed with error", "function", "Keys", "struct", "MariaDatabase", "namespace", namespace, "error", err)
		return nil, err
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			logger.Error("Scan row failed with error", "function", "Keys", "struct", "MariaDatabase", "namespace", namespace, "error", err)
			return keys, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (MDB *MariaDatabase) DeleteKey(namespace string, key string) error {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	stmt, err := MDB.Connection.Prepare(fmt.Sprintf("delete from `%v` where `%v` = ?", namespace, MDB.Config.KeyName))
	if err != nil {
		logger.Error("Prepare failed with error", "function", "DeleteKey", "struct", "MariaDatabase", "namespace", namespace, "key", key, "error", err)
		return err
	}
	_, err = stmt.Exec(key)
	if err != nil {
		logger.Error("Exec failed with error", "function", "DeleteKey", "struct", "MariaDatabase", "namespace", namespace, "key", key, "error", err)
		return err
	}
	return nil
}

func (MDB *MariaDatabase) CreateNamespace(namespace string) error {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	result, err := MDB.Connection.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` ( `%v` CHAR(%v) PRIMARY KEY, `%v` VARCHAR(%v) NOT NULL) ENGINE = InnoDB; ", namespace, MDB.Config.KeyName, rest.KeyMaxLength, MDB.Config.ValueName, rest.ValueMaxLength))
	logger.Debug("Create table if not exists", "function", "createTable", "struct", "MariaDatabase", "namespace", namespace, "result", result)
	if err != nil {
		logger.Error("Error creating table", "function", "createTable", "struct", "MariaDatabase", "namespace", namespace, "error", err)
		return err
	}
	return nil
}

func (MDB *MariaDatabase) DeleteNamespace(namespace string) error {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	if namespace == MDB.GetSystemNS() {
		return &ErrNotAllowed{Value: fmt.Sprintf("delete System NS %v", namespace)}
	}
	_, err := MDB.Connection.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%v`", namespace))
	if err != nil {
		logger.Error("Exec failed with error", "function", "Delete", "struct", "MariaDatabase", "namespace", namespace, "error", err)
		return err
	}
	return nil
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
