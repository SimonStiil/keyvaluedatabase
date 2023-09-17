package main

import (
	"testing"
)

type MariaDBTest struct {
	DB     Database
	Config Config
}

func Test_Maria_DB(t *testing.T) {
	dbt := new(MariaDBTest)
	dbt.Config = ConfigRead("example-config.yaml")
	if dbt.Config.Mysql.Address == "" {
		t.Log("no MySQL configuration in test config. Skipping test")
		return
	}
	dbt.DB = &MariaDatabase{
		DatabaseName: dbt.Config.Mysql.DatabaseName,
		TableName:    "kvdb-test",
		Host:         dbt.Config.Mysql.Address,
		User:         dbt.Config.Mysql.Username,
		Password:     dbt.Config.Mysql.Password,
	}
	t.Run("initialize db", func(t *testing.T) {
		dbt.DB.Init()
	})
	testKey := "test"
	testValue := "value"

	t.Run("Delete Key (if it exists)", func(t *testing.T) {
		dbt.DB.Delete(testKey)
	})

	t.Run("get value (that don't exist)", func(t *testing.T) {
		val, ok := dbt.DB.Get(testKey)
		if ok {
			t.Errorf("Supposed to not key %v", testKey)
		}
		if val != "" {
			t.Errorf("Read from database failed expected %v, got %v", "", val)
		}
	})
	t.Run("set value", func(t *testing.T) {
		dbt.DB.Set(testKey, testValue)
	})

	t.Run("get value", func(t *testing.T) {
		val, ok := dbt.DB.Get(testKey)
		if !ok {
			t.Errorf("Supposed to contain key %v", testKey)
		}
		if testValue != val {
			t.Errorf("Read from database failed expected %v, got %v", testValue, val)
		}
	})
	t.Run("Counter Integration Test (stored db)", func(t *testing.T) {
		var count Counter
		dbt.DB.Delete("counter")
		count.Init(dbt.DB)
		val := count.GetCount()
		if val != 0 {
			t.Errorf("Fresh Counter expected value to be 0, got %v", val)
		}
		val = count.GetCount()
		if val != 1 {
			t.Errorf("Counter expected value to be 1, got %v", val)
		}
		dbt.DB.Close()
		dbt.DB.Init()
		val = count.GetCount()
		if val != 2 {
			t.Errorf("Fresh Counter expected value to be 2, got %v", val)
		}
		val = count.GetCount()
		if val != 3 {
			t.Errorf("Counter expected value to be 3, got %v", val)
		}
	})
}
