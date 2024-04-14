package main

import (
	"reflect"
	"strconv"
	"sync"
)

type Counter struct {
	Mutex     sync.Mutex
	Value     uint32
	DB        Database
	testing   bool
	namespace string
}

func (Count *Counter) Init(DB Database) {
	logger.Debug("Initializing", "function", "Init", "struct", "Counter")
	Count.DB = DB
	Count.Mutex.Lock()
	defer Count.Mutex.Unlock()
	if Count.DB != nil && Count.DB.IsInitialized() {
		Count.namespace = Count.DB.GetSystemNS()
		val, ok := Count.DB.Get(Count.namespace, "counter")
		Count.Value = 0
		logger.Debug("Get count from db", "function", "Init", "struct", "Counter", "value", val, "type", reflect.TypeOf(val))
		if ok {
			fromString, err := strconv.ParseInt(val, 10, 32)
			if err != nil {
				panic(err)
			}
			Count.Value = uint32(fromString)
		}
	} else {
		Count.Value = 0
	}
	logger.Debug("Initialization complete", "function", "Init", "struct", "Counter")
}
func (Count *Counter) GetCount() uint32 {
	Count.Mutex.Lock()
	defer Count.Mutex.Unlock()
	var currentCount = Count.Value
	Count.Value = Count.Value + 1
	if Count.DB != nil && Count.DB.IsInitialized() {
		Count.DB.Set(Count.namespace, "counter", Count.Value)
	} else {
		if !Count.testing {
			logger.Error("Not initialized", "function", "GetCount", "struct", "Counter")
		}
	}
	return currentCount
}
func (Count *Counter) PeakCount() uint32 {
	Count.Mutex.Lock()
	defer Count.Mutex.Unlock()
	return Count.Value
}
