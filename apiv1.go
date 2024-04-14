package main

import (
	"encoding/json"
	"net/http"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	/*
		"crypto/tls"
		"crypto/x509"
		"errors"
		"fmt"
		"io"
		"log"
		"net/http"
		"os"
		"reflect"
		"runtime"
		"strings"

		"github.com/gorilla/schema"
	*/)

type APIv1 struct {
}

type APIv1Type uint

const (
	List     APIv1Type = 0
	FullList APIv1Type = 1
	Key      APIv1Type = 2
	Error    APIv1Type = 4
)

func (Api *APIv1) APIPrefix() string {
	return "v1"
}

func (api *APIv1) ApiController(w http.ResponseWriter, request *RequestParameters) {
	if request.Method == "UPDATE" || request.Method == "PATCH" {
		data := rest.KVUpdateV2{}
		err := App.decodeAny(request.orgRequest, &data)
		if err != nil {
			logger.Info("Unable to decodeAny",
				"function", "ApiController", "struct", "APIv1",
				"id", request.ID, "error", err)
		}
		request.AttachmentUpdate = &data
	} else {
		data := rest.KVPairV2{}
		err := App.decodeAny(request.orgRequest, &data)
		if err != nil {
			logger.Info("Unable to decodeAny",
				"function", "ApiController", "struct", "APIv1",
				"id", request.ID, "error", err)
		}
		request.AttachmentPair = &data
	}
	switch api.GetRequestType(request) {
	case FullList:
		api.fullList(w, request)
	case List:
		api.list(w, request)
	case Key:
		api.key(w, request)
	}
}

func (api *APIv1) GetRequestType(request *RequestParameters) APIv1Type {
	if request.AttachmentPair != nil {
		if request.Namespace == "" && request.AttachmentPair.Namespace != "" {
			request.Namespace = request.AttachmentPair.Namespace
		} else {
			if request.Namespace != "" && request.AttachmentPair.Namespace != "" &&
				request.Namespace != request.AttachmentPair.Namespace {
				return Error
			}
		}
		if request.Key == "" && request.AttachmentPair.Key != "" {
			request.Key = request.AttachmentPair.Key
		} else {
			if request.Key != "" && request.AttachmentPair.Key != "" &&
				request.Key != request.AttachmentPair.Key {
				return Error
			}
		}
	}

	if request.AttachmentUpdate != nil {
		if request.Namespace == "" && request.AttachmentUpdate.Namespace != "" {
			request.Namespace = request.AttachmentUpdate.Namespace
		} else {
			if request.Namespace != "" && request.AttachmentUpdate.Namespace != "" &&
				request.Namespace != request.AttachmentUpdate.Namespace {
				return Error
			}
		}
		if request.Key == "" && request.AttachmentUpdate.Key != "" {
			request.Key = request.AttachmentUpdate.Key
		} else {
			if request.Key != "" && request.AttachmentUpdate.Key != "" &&
				request.Key != request.AttachmentUpdate.Key {
				return Error
			}
		}
	}
	if len(request.Namespace) > 0 && request.Key == "*" {
		return FullList
	}
	if request.Namespace == "" || request.Key == "" {
		return List
	}
	return Key
}

func (api *APIv1) fullList(w http.ResponseWriter, request *RequestParameters) {
	logger.Info("Full List Request",
		"function", "list", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "status", 200)
	content := App.DB.Keys(request.Namespace)
	var fullList []rest.KVPairV2
	for _, key := range content {
		value, ok := App.DB.Get(request.Namespace, key)
		if ok {
			fullList = append(fullList, rest.KVPairV2{Key: key, Namespace: request.Namespace, Value: value})
		} else {
			logger.Info("Error reading key from db",
				"function", "list", "struct", "APIv1",
				"id", request.ID, "key", key)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullList)
}

func (api *APIv1) list(w http.ResponseWriter, request *RequestParameters) {
	logger.Info("List Request",
		"function", "list", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "status", 200)
	content := App.DB.Keys(request.Namespace)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(content)
}

func (api *APIv1) key(w http.ResponseWriter, request *RequestParameters) {
	logger.Debug("key Request - Start",
		"function", "key", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace)
	switch request.Method {
	case "GET":

		value, ok := App.DB.Get(request.Namespace, request.Key)
		logger.Debug("key Request - DB.Get",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "ok", ok, "value", value)
		if !ok {
			keys.WithLabelValues(request.Key, request.Namespace, "GET", "NotFound").Inc()
			http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
			return
		}
		reply := rest.KVPairV2{Key: request.Key, Namespace: request.Namespace, Value: value}
		keys.WithLabelValues(request.Key, request.Namespace, "GET", "OK").Inc()

		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 200)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply)
		return
	case "POST":
		if request.AttachmentPair == nil {
			keys.WithLabelValues(request.Key, request.Namespace, "POST", "BadRequest").Inc()
			App.BadRequestHandler(w, request)
		}
		App.DB.Set(request.Namespace, request.Key, request.AttachmentPair.Value)
		value, ok := App.DB.Get(request.Namespace, request.Key)
		logger.Debug("key Request - DB.Get",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "ok", ok, "value", value)
		if !ok {
			keys.WithLabelValues(request.Key, request.Namespace, "POST", "NotFound").Inc()
			http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
			return
		}
		keys.WithLabelValues(request.Key, request.Namespace, "POST", "OK").Inc()
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 201)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
		return

	case "PUT":
		App.DB.Set(request.Namespace, request.Key, request.AttachmentPair.Value)
		value, ok := App.DB.Get(request.Namespace, request.Key)
		logger.Debug("key Request - DB.Get",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "ok", ok, "value", value)
		if !ok {
			keys.WithLabelValues(request.Namespace, request.Key, "NotFound", "OK").Inc()
			http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
			return
		}
		keys.WithLabelValues(request.Namespace, request.Key, "PUT", "OK").Inc()
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 201)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
		return
	case "UPDATE", "PATCH":
		logger.Debug("PUT Request extras",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "metod", request.Method,
			"update.key", request.AttachmentUpdate.Key, "update.type", request.AttachmentUpdate.Type)

		newData := rest.KVPairV2{Key: request.Key, Namespace: request.Namespace, Value: AuthGenerateRandomString(32)}

		logger.Debug("UPDATE random",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "metod", request.Method,
			"newData.key", newData.Key, "newData.value", newData.Value)
		_, exists := App.DB.Get(request.Namespace, request.Key)
		if request.AttachmentUpdate.Type == rest.TypeRoll && exists {
			App.DB.Set(newData.Namespace, newData.Key, newData.Value)
			keys.WithLabelValues(request.Namespace, request.Key, "ROLL", "OK").Inc()

			logger.Info("key Request",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "address", request.RequestIP,
				"user", request.GetUserName(), "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace,
				"key", request.Key, "type", request.AttachmentUpdate.Type, "status", 200)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)
			return
		}
		if request.AttachmentUpdate.Type == rest.TypeGenerate && !exists {
			App.DB.Set(newData.Namespace, newData.Key, newData.Value)
			keys.WithLabelValues(request.Namespace, request.Key, "GENERATE", "OK").Inc()
			logger.Info("key Request",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "address", request.RequestIP,
				"user", request.GetUserName(), "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace,
				"key", request.Key, "type", request.AttachmentUpdate.Type, "status", 200)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)
			return
		}
		keys.WithLabelValues(request.Namespace, request.Key, "UPDATE", "BadRequest").Inc()
		App.BadRequestHandler(w, request)
		return
	case "DELETE":
		App.DB.Delete(request.Namespace, request.Key)
		keys.WithLabelValues(request.Namespace, request.Key, "DELETE", "OK").Inc()
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 200)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	default:
		http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
		return
	}
}

func (api *APIv1) Permissions(request *RequestParameters) *ConfigPermissions {
	switch api.GetRequestType(request) {
	case FullList:
		return &ConfigPermissions{List: true, Read: true}
	case List:
		return &ConfigPermissions{List: true}
	case Key:
		switch request.Method {
		case "GET":
			return &ConfigPermissions{Read: true}
		case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
			return &ConfigPermissions{Write: true}
		}
	}
	return &ConfigPermissions{Write: true, Read: true, List: true}
}
