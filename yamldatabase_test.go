package main

import (
	"errors"
	"os"
	"testing"
)

type DBTest struct {
	DB       Database
	FileName string
}

func Test_Yaml_DB(t *testing.T) {
	dbt := new(DBTest)
	setupTestlogging()
	dbt.FileName = "testdb.yaml"
	dbt.DB = &YamlDatabase{DatabaseName: dbt.FileName}
	t.Run("initialize fresh db", func(t *testing.T) {
		err := os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init()
	})
	testKey := "test"
	testValue := "value"

	t.Run("set value", func(t *testing.T) {
		dbt.DB.Set(dbt.DB.GetSystemNS(), testKey, testValue)
	})
	t.Run("get value (that don't exist yet)", func(t *testing.T) {
		_, err := dbt.DB.Get(dbt.DB.GetSystemNS(), testKey+"13")
		if err == nil {
			t.Errorf("Supposed to get error")
		}
		if _, ok := err.(*ErrNotFound); !ok {
			t.Errorf("Supposed to get ErrNotFound error got %v if tyoe %t", err, err)
		}
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
		err := os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init()
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
	t.Run("Counter Integration Test (fresh db)", func(t *testing.T) {
		count := Counter{}
		err := os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init()
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
		err = os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init()
		count.Init(dbt.DB)
		val = count.GetCount()
		if val != 0 {
			t.Errorf("Fresh Counter expected value to be 1, got %v", val)
		}
		val = count.GetCount()
		if val != 1 {
			t.Errorf("Counter expected value to be 0, got %v", val)
		}
	})
}
