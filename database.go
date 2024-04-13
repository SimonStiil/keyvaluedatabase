package main

// https://gobyexample.com/interfaces

type Database interface {
	Init()
	Set(namespace string, key string, value interface{})
	Get(namespace string, key string) (string, bool)
	GetSystemNS() string
	Delete(namespace string, key string)
	Keys(namespace string) []string
	Close()
	IsInitialized() bool
}
