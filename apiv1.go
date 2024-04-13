package main

import (
	/*
		"crypto/tls"
		"crypto/x509"
		"encoding/json"
		"errors"
		"fmt"
		"io"
		"log"
		"net/http"
		"os"
		"reflect"
		"runtime"
		"strings"

		"github.com/SimonStiil/keyvaluedatabase/rest"
		"github.com/gorilla/schema"
	*/
	"net/http"
)

type APIv1 struct{}

func (api *APIv1) ApiController(w http.ResponseWriter, r *http.Request) {

}
func (api *APIv1) Permissions(request *RequestParameters) *ConfigPermissions {
	if len(request.Secret) == 0 {
		return &ConfigPermissions{List: true}
	}
	switch request.Method {
	case "GET":
		return &ConfigPermissions{Read: true}
	case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
		return &ConfigPermissions{Write: true}
	default:
		return &ConfigPermissions{Write: true, Read: true, List: true}
	}
}
