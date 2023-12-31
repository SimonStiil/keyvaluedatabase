package main

// https://gobyexample.com/interfaces

type Database interface {
	Init()
	Set(key string, value interface{})
	Get(key string) (string, bool)
	Delete(key string)
	Keys() []string
	Close()
	IsInitialized() bool
}
