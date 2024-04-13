package main

import "net/http"

// https://gobyexample.com/interfaces

type API interface {
	Permissions(request RequestParameters) *ConfigPermissions
	ApiController(w http.ResponseWriter, request *RequestParameters)
}
