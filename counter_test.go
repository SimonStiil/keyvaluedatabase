package main

import (
	"sync"
	"testing"
)

type CountTest struct {
	Count Counter
	Mutex sync.Mutex
}

func Test_getCount(t *testing.T) {
	ct := new(CountTest)
	ct.Count.testing = true
	setupTestlogging()
	ct.GetCountTest(t)
}

func (CT *CountTest) GetCountTest(t *testing.T) {
	t.Run("Race condition Test", func(t *testing.T) {
		output := make(chan uint32)
		go CT.GetCount_Tester(output, 10)
		go CT.GetCount_Tester(output, 10)
		go CT.GetCount_Tester(output, 10)
		go CT.GetCount_Tester(output, 10)
		for i := 0; i < 40; i++ {
			value := <-output
			if i != int(value) {
				t.Errorf("getCount() = %v, want %v", value, i)
			}
		}
	})
}

func (CT *CountTest) GetCount_Tester(output chan<- uint32, reads int) {
	for i := 0; i < reads; i++ {
		CT.Mutex.Lock()
		output <- CT.Count.GetCount()
		CT.Mutex.Unlock()
	}
}
