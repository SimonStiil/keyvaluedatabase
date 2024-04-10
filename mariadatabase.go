package main

import (
	"fmt"
	"log"
	"os"

	"database/sql"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

type MariaDatabase struct {
	Initialized  bool
	Connection   *sql.DB
	Config       *ConfigType
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
	configReader.SetDefault("mysql.address", "localhost:3306")
	configReader.SetDefault("mysql.username", "kvdb")
	configReader.SetDefault("mysql.databaseName", "")
	configReader.SetDefault("mysql.tableName", "kvdb")
	configReader.SetDefault("mysql.envVariableName", BaseENVname+"_MYSQL_PASSWORD")
	configReader.SetDefault("mysql.keyName", "key")
	configReader.SetDefault("mysql.valueName", "value")
}

func (MDB *MariaDatabase) Init() {
	if MDB.Config.Debug {
		log.Println("D db.init (MariaDB)")
	}
	if MDB.Config.Mysql.DatabaseName == "" {
		MDB.DatabaseName = MDB.Config.Mysql.Username
	} else {
		MDB.DatabaseName = MDB.Config.Mysql.DatabaseName
	}
	MDB.Password = os.Getenv(MDB.Config.Mysql.EnvVariableName)
	var err error
	connectionString := fmt.Sprintf("%v:%v@tcp(%v)/%v", MDB.Config.Mysql.Username, MDB.Password, MDB.Config.Mysql.Address, MDB.DatabaseName)

	if MDB.Config.Debug {
		log.Println("D db.init - ", connectionString)
	}
	MDB.Connection, err = sql.Open("mysql", connectionString)
	if err != nil {
		panic(err.Error())
	}
	_, err = MDB.Connection.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` ( `%v` CHAR(32) PRIMARY KEY, `%v` VARCHAR(16000) NOT NULL) ENGINE = InnoDB; ", MDB.Config.Mysql.TableName, MDB.Config.Mysql.KeyName, MDB.Config.Mysql.ValueName))
	if err != nil {
		panic(err.Error())
	}
	if MDB.Config.Debug {
		log.Println("D db.init - complete")
	}
	MDB.Initialized = true
}

func (MDB *MariaDatabase) Set(key string, value interface{}) {
	if !MDB.Initialized {
		panic("F Unable to set. db not initialized()")
	}
	statement, err := MDB.Connection.Prepare(fmt.Sprintf("INSERT INTO `%v` (`%v`, `%v`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `%v`=?", MDB.Config.Mysql.TableName, MDB.Config.Mysql.KeyName, MDB.Config.Mysql.ValueName, MDB.Config.Mysql.ValueName))
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
	rows, err := MDB.Connection.Query(fmt.Sprintf("select * from `%v` where `%v` = ? ", MDB.Config.Mysql.TableName, MDB.Config.Mysql.KeyName), key)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var kvpair rest.KVPairV1
	found := false
	for rows.Next() {
		err = rows.Scan(&kvpair.Key, &kvpair.Value)
		if err != nil {
			log.Fatal(err)
		}
		found = true
	}
	return kvpair.Value, found
}

func (MDB *MariaDatabase) Keys() []string {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	rows, err := MDB.Connection.Query(fmt.Sprintf("select `%v` from `%v`", MDB.Config.Mysql.KeyName, MDB.Config.Mysql.TableName))
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil {
			log.Fatal(err)
		}
		keys = append(keys, key)
	}
	return keys
}

func (MDB *MariaDatabase) Delete(key string) {
	if !MDB.Initialized {
		panic("F Unable to get. db not initialized()")
	}
	stmt, err := MDB.Connection.Prepare(fmt.Sprintf("delete from `%v` where `%v` = ?", MDB.Config.Mysql.TableName, MDB.Config.Mysql.KeyName))
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(key)
	if err != nil {
		log.Fatal(err)
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
	if MDB.Config.Debug {
		log.Println("D db.closed")
	}
}
