package main

import (
	"log"
	"reflect"
	"strconv"
	"sync"
)

type Counter struct {
	Mutex  sync.Mutex
	Value  uint32
	Config *ConfigType
	DB     Database
}

func (Count *Counter) Init(DB Database) {
	if Count.Config.Debug {
		log.Println("D count.Init")
	}
	Count.DB = DB
	Count.Mutex.Lock()
	defer Count.Mutex.Unlock()
	if Count.DB != nil && Count.DB.IsInitialized() {
		val, ok := Count.DB.Get("counter")
		Count.Value = 0
		if Count.Config.Debug {
			log.Println("D count.Init get - ", val, " type: ", reflect.TypeOf(val))
		}
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
	if Count.Config.Debug {
		log.Println("D count.Init - complete")
	}
}
func (Count *Counter) GetCount() uint32 {
	Count.Mutex.Lock()
	defer Count.Mutex.Unlock()
	var currentCount = Count.Value
	Count.Value = Count.Value + 1
	if Count.DB != nil && Count.DB.IsInitialized() {
		Count.DB.Set("counter", Count.Value)
	} else {
		if Count.Config.Debug {
			log.Println("D getCount db.isInitialized not true")
		}
	}
	return currentCount
}
func (Count *Counter) PeakCount() uint32 {
	Count.Mutex.Lock()
	defer Count.Mutex.Unlock()
	return Count.Value
}
