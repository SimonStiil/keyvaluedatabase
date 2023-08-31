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
	dbt.DB = new(YamlDatabase)
	dbt.FileName = "testdb.yaml"
	t.Run("initialize fresh db", func(t *testing.T) {
		err := os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init(dbt.FileName, "")
	})
	testKey := "test"
	testValue := "value"

	t.Run("get value (that don't exist yet)", func(t *testing.T) {
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
		err := os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init(dbt.FileName, "")
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
		dbt.DB.Init(dbt.FileName, "")
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
		var count Counter
		err := os.Remove(dbt.FileName)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatal(err)
		}
		dbt.DB.Init(dbt.FileName, "")
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
		dbt.DB.Init(dbt.FileName, "")
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
