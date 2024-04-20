package main

import "fmt"

// https://gobyexample.com/interfaces

type Database interface {
	Init()
	Set(namespace string, key string, value interface{}) error
	Get(namespace string, key string) (string, error)
	GetSystemNS() string
	DeleteKey(namespace string, key string) error
	CreateNamespace(namespace string) error
	DeleteNamespace(namespace string) error
	Keys(namespace string) ([]string, error)
	Close()
	IsInitialized() bool
}

type ErrNotFound struct {
	Value string
}

func (err *ErrNotFound) Error() string {
	return fmt.Sprintf("%v not found", err.Value)
}
