package main

import (
	"testing"
)

type MariaDBTest struct {
	DB     Database
	Config ConfigType
}

func Test_Maria_DB(t *testing.T) {
	dbt := new(MariaDBTest)
	setupTestlogging()
	ConfigRead("example-config", &dbt.Config)
	if dbt.Config.DatabaseType != "mysql" {
		t.Log("no MySQL configuration in test config. Skipping test")
		return
	}
	dbt.Config.Mysql.DatabaseName = "kvdb-test"
	dbt.DB = &MariaDatabase{
		Config: &dbt.Config.Mysql,
	}
	t.Run("initialize db", func(t *testing.T) {
		dbt.DB.Init()
	})
	testKey := "test"
	testValue := "value"

	t.Run("Delete Key (if it exists)", func(t *testing.T) {
		dbt.DB.DeleteKey(dbt.DB.GetSystemNS(), testKey)
	})

	t.Run("get value (that don't exist)", func(t *testing.T) {
		val, err := dbt.DB.Get(dbt.DB.GetSystemNS(), testKey)
		if err != nil {
			t.Errorf("Supposed to get key %v got error %+v", testKey, err)
		}
		if val != "" {
			t.Errorf("Read from database failed expected %v, got %v", "", val)
		}
	})
	t.Run("set value", func(t *testing.T) {
		dbt.DB.Set(dbt.DB.GetSystemNS(), testKey, testValue)
	})

	t.Run("get value", func(t *testing.T) {
		val, err := dbt.DB.Get(dbt.DB.GetSystemNS(), testKey)
		if err != nil {
			t.Errorf("Supposed to get key %v got error %+v", testKey, err)
		}
		if testValue != val {
			t.Errorf("Read from database failed expected %v, got %v", testValue, val)
		}
	})
	t.Run("Counter Integration Test (stored db)", func(t *testing.T) {
		count := Counter{}
		dbt.DB.DeleteKey(dbt.DB.GetSystemNS(), "counter")
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
