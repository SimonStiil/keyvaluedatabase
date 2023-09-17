package main

import (
	"fmt"
	"log"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type MariaDatabase struct {
	Initialized  bool
	Connection   *sql.DB
	Host         string
	User         string
	Password     string
	DatabaseName string
	TableName    string
	KeyName      string
	ValueName    string
}

func (MDB *MariaDatabase) Init() {
	if debug {
		log.Println("db.init (MariaDB)")
	}
	if MDB.User == "" {
		MDB.User = "kvdb"
	}
	if MDB.DatabaseName == "" {
		MDB.DatabaseName = MDB.User
	}
	if MDB.TableName == "" {
		MDB.TableName = "kvdb"
	}
	if MDB.KeyName == "" {
		MDB.KeyName = "key"
	}
	if MDB.ValueName == "" {
		MDB.ValueName = "value"
	}
	var err error
	connectionString := fmt.Sprintf("%v:%v@tcp(%v)/%v", MDB.User, MDB.Password, MDB.Host, MDB.DatabaseName)

	log.Println("db.init - ", connectionString)
	MDB.Connection, err = sql.Open("mysql", connectionString)
	if err != nil {
		panic(err.Error())
	}
	MDB.Connection.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%v` ( `%v` CHAR(32) PRIMARY KEY, `%v` VARCHAR(21800) NOT NULL) ENGINE = InnoDB; ", MDB.TableName, MDB.KeyName, MDB.ValueName))
	if debug {
		log.Println("db.init - complete")
	}
	MDB.Initialized = true
}

func (MDB *MariaDatabase) Set(key string, value interface{}) {
	if !MDB.Initialized {
		panic("Unable to set. db not initialized()")
	}
	statement, err := MDB.Connection.Prepare(fmt.Sprintf("INSERT INTO `%v` (`%v`, `%v`) VALUES (?, ?) ON DUPLICATE KEY UPDATE `%v`=VALUES(`%v`)", MDB.TableName, MDB.KeyName, MDB.ValueName, MDB.KeyName, MDB.KeyName))
	if err != nil {
		panic(err)
	}
	_, err = statement.Exec(key, value)
	if err != nil {
		panic(err)
	}
}

func (MDB *MariaDatabase) Get(key string) (string, bool) {
	if !MDB.Initialized {
		panic("Unable to get. db not initialized()")
	}
	rows, err := MDB.Connection.Query(fmt.Sprintf("select * from `%v` where `%v` = ? ", MDB.TableName, MDB.KeyName), key)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var kvpair KVPair
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
		panic("Unable to get. db not initialized()")
	}
	rows, err := MDB.Connection.Query(fmt.Sprintf("select `%v` from `%v`", MDB.KeyName, MDB.TableName))
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
		panic("Unable to get. db not initialized()")
	}
	stmt, err := MDB.Connection.Prepare(fmt.Sprintf("delete from `%v` where `%v` = ?", MDB.TableName, MDB.KeyName))
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
		panic("Unable to close. db not initialized()")
	}
	err := MDB.Connection.Close()
	if err != nil {
		panic(err)
	}
	if debug {
		log.Println("db.closed")
	}
}
