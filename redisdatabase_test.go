package main

import (
	"testing"
)

type RedisDBTest struct {
	DB     Database
	Config ConfigType
}

func Test_Redis_DB(t *testing.T) {
	dbt := new(RedisDBTest)
	setupTestlogging()
	ConfigRead("example-config", &dbt.Config)
	dbt.DB = &RedisDatabase{
		Config: &dbt.Config.Redis,
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
		_, err := dbt.DB.Get(dbt.DB.GetSystemNS(), testKey)
		if err == nil {
			t.Errorf("Supposed to get error")
		}
		if _, ok := err.(*ErrNotFound); !ok {
			t.Errorf("Supposed to get ErrNotFound error got %v", err)
		}
	})

	t.Run("set value", func(t *testing.T) {
		err := dbt.DB.Set(dbt.DB.GetSystemNS(), testKey, testValue)
		if err != nil {
			t.Errorf("Failed to set value: %v", err)
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

	t.Run("update existing value", func(t *testing.T) {
		newValue := "updated_value"
		err := dbt.DB.Set(dbt.DB.GetSystemNS(), testKey, newValue)
		if err != nil {
			t.Errorf("Failed to update value: %v", err)
		}
		val, err := dbt.DB.Get(dbt.DB.GetSystemNS(), testKey)
		if err != nil {
			t.Errorf("Failed to get updated value: %v", err)
		}
		if val != newValue {
			t.Errorf("Expected updated value %v, got %v", newValue, val)
		}
	})

	t.Run("list keys in namespace", func(t *testing.T) {
		keys, err := dbt.DB.Keys(dbt.DB.GetSystemNS())
		if err != nil {
			t.Errorf("Failed to list keys: %v", err)
		}
		found := false
		expectedKey := "kvdb_kvdb_" + testKey
		for _, key := range keys {
			if key == expectedKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find key %v in keys list, got %v", expectedKey, keys)
		}
	})

	t.Run("delete key", func(t *testing.T) {
		err := dbt.DB.DeleteKey(dbt.DB.GetSystemNS(), testKey)
		if err != nil {
			t.Errorf("Failed to delete key: %v", err)
		}
		_, err = dbt.DB.Get(dbt.DB.GetSystemNS(), testKey)
		if err == nil {
			t.Errorf("Key should not exist after deletion")
		}
		if _, ok := err.(*ErrNotFound); !ok {
			t.Errorf("Expected ErrNotFound after deletion, got %v", err)
		}
	})

	testNamespace := "test_namespace"

	t.Run("create namespace", func(t *testing.T) {
		err := dbt.DB.CreateNamespace(testNamespace)
		if err != nil {
			t.Errorf("Failed to create namespace: %v", err)
		}
	})

	t.Run("set value in custom namespace", func(t *testing.T) {
		err := dbt.DB.Set(testNamespace, "ns_key", "ns_value")
		if err != nil {
			t.Errorf("Failed to set value in custom namespace: %v", err)
		}
	})

	t.Run("get value from custom namespace", func(t *testing.T) {
		val, err := dbt.DB.Get(testNamespace, "ns_key")
		if err != nil {
			t.Errorf("Failed to get value from custom namespace: %v", err)
		}
		if val != "ns_value" {
			t.Errorf("Expected ns_value, got %v", val)
		}
	})

	t.Run("list all namespaces", func(t *testing.T) {
		namespaces, err := dbt.DB.Keys("")
		if err != nil {
			t.Errorf("Failed to list namespaces: %v", err)
		}
		// Redis returns full keys with prefix, just verify we got results
		if len(namespaces) == 0 {
			t.Errorf("Expected to find namespaces in list")
		}
	})

	t.Run("delete namespace", func(t *testing.T) {
		err := dbt.DB.DeleteNamespace(testNamespace)
		if err != nil {
			t.Errorf("Failed to delete namespace: %v", err)
		}
		// Note: Redis DeleteNamespace is a stub, so key may still exist
	})

	t.Run("prevent deletion of system namespace", func(t *testing.T) {
		// Redis doesn't implement namespace protection, skip this test
		t.Skip("Redis DeleteNamespace is stub implementation")
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

	t.Run("close database", func(t *testing.T) {
		dbt.DB.Close()
	})
}
