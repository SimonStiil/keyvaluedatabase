package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type APIv1 struct {
}

type APIv1Type uint

const (
	List               APIv1Type = 0
	FullListKeys       APIv1Type = 1
	Key                APIv1Type = 2
	Error              APIv1Type = 4
	FullListNamespaces APIv1Type = 5
	Namespace          APIv1Type = 6
)

func (Api *APIv1) APIPrefix() string {
	return "v1"
}

func (api *APIv1) ApiController(w http.ResponseWriter, request *RequestParameters) {
	if request.Method == "UPDATE" || request.Method == "PATCH" || request.Method == "POST" || request.Method == "PUT" {
		data := rest.ObjectV1{}
		err := App.decodeAny(request.orgRequest, &data)
		if err != nil {
			logger.Info("Unable to decodeAny",
				"function", "ApiController", "struct", "APIv1",
				"id", request.ID, "error", err)
		}
		request.Attachment = &data
	}
	logger.Debug("ApiController",
		"function", "ApiController", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"namespace", request.Namespace, "key", request.Key,
		"attachment", request.Attachment)
	switch api.GetRequestType(request) {
	case FullListKeys:
		api.fullListKeys(w, request)
	case List:
		api.list(w, request)
	case FullListNamespaces:
		api.fullListNamespaces(w, request)
	case Key:
		api.key(w, request)
	case Namespace:
		api.namespace(w, request)
	default:
		logger.Error("Handler Not found",
			"function", "ApiController", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 404)
		http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
	}
}

func (api *APIv1) GetRequestType(request *RequestParameters) APIv1Type {
	if request.Attachment != nil {
		switch request.Attachment.Type {
		case rest.TypeKey, rest.TypeRoll, rest.TypeGenerate:
			return Key
		case rest.TypeNamespace:
			return Namespace
		}
	}
	if request.Method == "GET" && request.Namespace == "*" && request.Key == "" {
		return FullListNamespaces
	}
	if request.Method == "GET" && len(request.Namespace) > 0 && request.Key == "*" {
		return FullListKeys
	}
	if request.Method == "GET" && (request.Namespace == "" || request.Key == "") {
		return List
	}
	if len(request.Namespace) > 0 && len(request.Key) > 0 {
		return Key
	}
	if len(request.Namespace) > 0 && len(request.Key) == 0 {
		return Namespace
	}
	return Error
}

func (api *APIv1) fullListKeys(w http.ResponseWriter, request *RequestParameters) {
	status := http.StatusOK
	logger.Info("Full List Keys Request",
		"function", "list", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "status", 200)
	content, err := App.DB.Keys(request.Namespace)
	if err != nil {
		status = http.StatusInternalServerError
		logger.Debug("Error listing keys from db",
			"function", "fullListKeys", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace, "Error", err)
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	}
	var fullList rest.KVPairListV1
	for _, key := range content {
		value, err := App.DB.Get(request.Namespace, key)
		if err == nil {
			fullList = append(fullList, rest.KVPairV2{Key: key, Namespace: request.Namespace, Value: value})
		} else {
			status = http.StatusInternalServerError
			logger.Debug("Error reading key from db",
				"function", "fullListKeys", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "key", key, "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullList)
}

func (api *APIv1) fullListNamespaces(w http.ResponseWriter, request *RequestParameters) {
	status := http.StatusOK
	logger.Info("Full List Namespaces Request",
		"function", "list", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "status", 200)
	content, err := App.DB.Keys("")
	if err != nil {
		status = http.StatusInternalServerError
		logger.Debug("Error listing namespaces from db",
			"function", "fullListNamespaces", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace, "Error", err)
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	}
	requestOrgNamespace := request.Namespace
	testPermissons := ConfigPermissions{Read: true, List: true, Write: false}
	user := request.Authentication.User
	var fullList []rest.NamespaceV2
	// TODO: Optimize for mysql
	for _, namespace := range content {
		keyslist, err := App.DB.Keys(namespace)
		if err != nil {
			status = http.StatusInternalServerError
			logger.Debug("Error listing keys from db",
				"function", "fullListNamespaces", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		request.Namespace = namespace
		access := user.Autorization(request, &testPermissons)
		fullList = append(fullList, rest.NamespaceV2{Name: namespace, Size: len(keyslist), Access: access})
	}
	request.Namespace = requestOrgNamespace
	logger.Info("Full List Namespaces", "reply", fullList)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullList)
}

func (api *APIv1) list(w http.ResponseWriter, request *RequestParameters) {
	logger.Info("List Request",
		"function", "list", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "status", 200)
	content, err := App.DB.Keys(request.Namespace)
	if err != nil {
		logger.Debug("Error listing from db",
			"function", "list", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace, "Error", err)
		status := http.StatusInternalServerError
		if _, ok := err.(*ErrNotFound); ok {
			status = http.StatusNotFound
		}
		keys.WithLabelValues(request.Key, request.Namespace, "GET", App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(http.StatusInternalServerError, w, request)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(content)
}

func (api *APIv1) key(w http.ResponseWriter, request *RequestParameters) {
	status := http.StatusOK
	logger.Debug("key Request - Start",
		"function", "key", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace)
	switch request.Method {
	case "GET":
		value, err := App.DB.Get(request.Namespace, request.Key)
		if err != nil {
			status = http.StatusInternalServerError
			logger.Debug("Error getting key from db",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
			if _, ok := err.(*ErrNotFound); ok {
				status = http.StatusNotFound
			}
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		logger.Debug("key Request - DB.Get",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "value", value)
		reply := rest.KVPairV2{Key: request.Key, Namespace: request.Namespace, Value: value}
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()

		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", status)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply)
		return
	case "POST":
		if request.Attachment == nil {
			keys.WithLabelValues(request.Key, request.Namespace, "POST", "BadRequest").Inc()
			App.WriteStatusMessage(http.StatusBadRequest, w, request)
		}
		err := App.DB.Set(request.Namespace, request.Key, request.Attachment.Value)
		if err != nil {
			logger.Debug("Error setting key in db",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
			status := http.StatusInternalServerError
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		value, err := App.DB.Get(request.Namespace, request.Key)
		if err != nil {
			logger.Debug("Error getting key from db",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
			status := http.StatusInternalServerError
			if _, ok := err.(*ErrNotFound); ok {
				status = http.StatusNotFound
			}
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		status = http.StatusCreated
		logger.Debug("key Request - DB.Get",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "value", value)
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", status)
		App.WriteStatusMessage(status, w, request)
		return

	case "PUT":
		App.DB.Set(request.Namespace, request.Key, request.Attachment.Value)
		value, err := App.DB.Get(request.Namespace, request.Key)
		if err != nil {
			logger.Debug("Error getting key from db",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
			status := http.StatusInternalServerError
			if _, ok := err.(*ErrNotFound); ok {
				status = http.StatusNotFound
			}
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		status = http.StatusCreated
		logger.Debug("key Request - DB.Get",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "value", value)
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", status)
		App.WriteStatusMessage(status, w, request)
		return
	case "UPDATE", "PATCH":
		logger.Debug("PUT Request extras",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "metod", request.Method,
			"update.type", request.Attachment.Type,
			"update.value", request.Attachment.Value)
		newKey := request.Key
		if request.Key == "" { // Should never happen anymore :-/
			newKey = AuthGenerateRandomString(16)
		}
		newData := rest.KVPairV2{Key: newKey, Namespace: request.Namespace, Value: AuthGenerateRandomString(32)}

		logger.Debug("UPDATE random",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "namespace", request.Namespace,
			"key", request.Key, "metod", request.Method,
			"newData.key", newData.Key, "newData.value", newData.Value)
		_, err := App.DB.Get(request.Namespace, request.Key)
		exists := err != nil
		if !exists {
			if !errors.Is(err, &ErrNotFound{}) {
				status = http.StatusInternalServerError
				logger.Debug("Error getting key from db",
					"function", "key", "struct", "APIv1",
					"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
				App.WriteStatusMessage(status, w, request)
				return
			}
		}
		if request.Attachment.Type == rest.TypeRoll && exists {
			err := App.DB.Set(newData.Namespace, newData.Key, newData.Value)
			if err != nil {
				logger.Debug("Error setting key in db",
					"function", "key", "struct", "APIv1",
					"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
				status := http.StatusInternalServerError
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
				App.WriteStatusMessage(status, w, request)
				return
			}
			keys.WithLabelValues(request.Namespace, request.Key, "ROLL", "OK").Inc()

			logger.Info("key Request",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "address", request.RequestIP,
				"user", request.GetUserName(), "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace,
				"key", request.Key, "type", request.Attachment.Type, "status", status)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)
			return
		}
		if request.Attachment.Type == rest.TypeGenerate && !exists {
			err := App.DB.Set(newData.Namespace, newData.Key, newData.Value)
			if err != nil {
				logger.Debug("Error setting key in db",
					"function", "key", "struct", "APIv1",
					"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
				status := http.StatusInternalServerError
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
				App.WriteStatusMessage(status, w, request)
				return
			}
			keys.WithLabelValues(request.Namespace, request.Key, "GENERATE", "OK").Inc()
			logger.Info("key Request",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "address", request.RequestIP,
				"user", request.GetUserName(), "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace,
				"key", request.Key, "type", request.Attachment.Type, "status", status)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)
			return
		}
		keys.WithLabelValues(request.Namespace, request.Key, "UPDATE", "BadRequest").Inc()
		App.WriteStatusMessage(http.StatusBadRequest, w, request)
		return
	case "DELETE":
		err := App.DB.DeleteKey(request.Namespace, request.Key)
		if err != nil {
			status := http.StatusInternalServerError
			logger.Debug("Error deleting key in db",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "namespace", request.Namespace, "key", request.Key, "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", status)
		App.WriteStatusMessage(status, w, request)
		return
	default:
		status = http.StatusNotFound
		logger.Info("key Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", status)
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	}
}

func (api *APIv1) namespace(w http.ResponseWriter, request *RequestParameters) {
	status := http.StatusOK
	logger.Debug("namespace Request - Start",
		"function", "namespace", "struct", "APIv1",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace,
		"attachment", request.Attachment)
	switch request.Method {
	case "POST":
		if request.Attachment == nil {
			keys.WithLabelValues(request.Key, request.Namespace, "POST", "BadRequest").Inc()
			App.WriteStatusMessage(http.StatusBadRequest, w, request)
		}
		namespace := request.Namespace
		if namespace == "" {
			namespace = request.Attachment.Value
		}
		if namespace == "" {
			keys.WithLabelValues(request.Key, request.Namespace, "POST", "BadRequest").Inc()
			App.WriteStatusMessage(http.StatusBadRequest, w, request)
		}
		error := App.DB.CreateNamespace(namespace)
		if error != nil {
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			logger.Debug("Create namespace Error",
				"function", "namespace", "struct", "APIv1",
				"id", request.ID, "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", namespace,
				"attachment", request.Attachment, "error", error)
			http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
			return
		}
		status = http.StatusCreated
		logger.Info("namespace Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", namespace, "key", request.Key, "status", status)
		keys.WithLabelValues(request.Key, namespace, request.Method, http.StatusText(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	case "DELETE":
		error := App.DB.DeleteNamespace(request.Namespace)
		if error != nil {
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			logger.Debug("Create namespace Error",
				"function", "namespace", "struct", "APIv1",
				"id", request.ID, "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace,
				"attachment", request.Attachment, "error", error)
			http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
			return
		}
		status = http.StatusOK
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		logger.Info("namespace Request",
			"function", "namespace", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 200)
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	default:
		logger.Info("namespace Request",
			"function", "key", "struct", "APIv1",
			"id", request.ID, "address", request.RequestIP,
			"user", request.GetUserName(), "method", request.Method,
			"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", 404)
		http.NotFoundHandler().ServeHTTP(w, request.orgRequest)
		return
	}
}

func (api *APIv1) Permissions(request *RequestParameters) *ConfigPermissions {
	switch api.GetRequestType(request) {
	case FullListKeys:
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
