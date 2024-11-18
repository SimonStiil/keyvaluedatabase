package main

import (
	"encoding/json"
	"net/http"

	"github.com/SimonStiil/keyvaluedatabase/rest"
)

type APIv1 struct {
}

type APIv1Type string

const (
	List               APIv1Type = "List"
	FullListKeys       APIv1Type = "FullListKeys"
	Key                APIv1Type = "Key"
	Error              APIv1Type = "Error"
	FullListNamespaces APIv1Type = "FullListNamespaces"
	Namespace          APIv1Type = "Namespace"
)

func (Api *APIv1) APIPrefix() string {
	return "v1"
}

func (api *APIv1) ApiController(w http.ResponseWriter, request *RequestParameters) {
	debugLogger := request.Logger.Ext.With("function", "ApiController", "struct", "APIv1")
	if request.Method == "UPDATE" || request.Method == "PATCH" || request.Method == "POST" || request.Method == "PUT" {
		data := rest.ObjectV1{}
		err := App.decodeAny(request, &data)
		if err != nil {
			request.Logger.Log.Error("Unable to decode data", "error", err)
			App.WriteStatusMessage(http.StatusBadRequest, w, request)
			return
		}
		request.Attachment = &data
	}
	requestType := api.GetRequestType(request)
	debugLogger.Debug("ApiController", "attachment", request.Attachment, "requestType", requestType)
	switch requestType {
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
		App.WriteStatusMessage(http.StatusNotFound, w, request)
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
	if request.Method == "POST" && len(request.Namespace) > 0 && len(request.Key) == 0 {
		return Key
	}
	if len(request.Namespace) > 0 && len(request.Key) == 0 {
		return Namespace
	}
	if request.Method == "POST" && len(request.Namespace) == 0 && len(request.Key) == 0 {
		return Namespace
	}
	return Error
}

func (api *APIv1) fullListKeys(w http.ResponseWriter, request *RequestParameters) {
	debugLogger := request.Logger.Ext.With("function", "fullListKeys")
	status := http.StatusOK
	debugLogger.Debug("Full List Keys Request")
	content, err := App.DB.Keys(request.Namespace)
	if err != nil {
		status = http.StatusInternalServerError
		debugLogger.Debug("Error listing keys from db", "Error", err)
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
			debugLogger.Debug("Error reading key from db", "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
	}
	request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", http.StatusText(status))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullList)
}

func (api *APIv1) fullListNamespaces(w http.ResponseWriter, request *RequestParameters) {
	debugLogger := request.Logger.Ext.With("function", "fullListNamespaces")
	status := http.StatusOK
	debugLogger.Debug("Full List Namespaces Request")
	content, err := App.DB.Keys("")
	if err != nil {
		status = http.StatusInternalServerError
		debugLogger.Debug("Error listing namespaces from db", "Error", err)
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
			debugLogger.Debug("Error listing keys from db", "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		request.Namespace = namespace
		access := user.Autorization(request, &testPermissons)
		fullList = append(fullList, rest.NamespaceV2{Name: namespace, Size: len(keyslist), Access: access})
	}
	request.Namespace = requestOrgNamespace
	debugLogger.Debug("Full List Namespaces", "reply", fullList)
	request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", http.StatusText(status))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullList)
}

func (api *APIv1) list(w http.ResponseWriter, request *RequestParameters) {
	status := http.StatusOK
	debugLogger := request.Logger.Ext.With("function", "list")
	content, err := App.DB.Keys(request.Namespace)
	if err != nil {
		debugLogger.Debug("Error listing from db", "Error", err)
		status := http.StatusInternalServerError
		if _, ok := err.(*ErrNotFound); ok {
			status = http.StatusNotFound
		}
		keys.WithLabelValues(request.Key, request.Namespace, "GET", App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(http.StatusInternalServerError, w, request)
		return
	}
	debugLogger.Debug("List", "reply", content)
	request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", http.StatusText(status))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(content)
}

func (api *APIv1) key(w http.ResponseWriter, request *RequestParameters) {
	debugLogger := request.Logger.Ext.With("function", "key")
	status := http.StatusOK
	debugLogger.Debug("key Request - Start")
	switch request.Method {
	case "GET":
		value, err := App.DB.Get(request.Namespace, request.Key)
		if err != nil {
			status = http.StatusInternalServerError
			debugLogger.Debug("Error getting key from db", "Error", err)

			if _, ok := err.(*ErrNotFound); ok {
				status = http.StatusNotFound
			}
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		debugLogger.Debug("key Request - DB.Get", "value", value)
		reply := rest.KVPairV2{Key: request.Key, Namespace: request.Namespace, Value: value}
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
		request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", http.StatusText(status))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply)
		return
	case "POST":
		if request.Attachment == nil {
			keys.WithLabelValues(request.Key, request.Namespace, "POST", "BadRequest").Inc()
			App.WriteStatusMessage(http.StatusBadRequest, w, request)
		}
		debugLogger.Debug("POST Content", "value", request.Attachment.Value, "type", request.Attachment.Type)
		err := App.DB.Set(request.Namespace, request.Key, request.Attachment.Value)
		if err != nil {
			debugLogger.Debug("Error setting key in db", "Error", err)
			status := http.StatusInternalServerError
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		// Should not be nessesary to test that object is created....
		/*
			value, err := App.DB.Get(request.Namespace, request.Key)
			if err != nil {
				debugLogger.Debug("Error getting key from db", "Error", err)
				status := http.StatusInternalServerError
				if _, ok := err.(*ErrNotFound); ok {
					status = http.StatusNotFound
				}
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
				App.WriteStatusMessage(status, w, request)
				return
			}
			status = http.StatusCreated
			debugLogger.Debug("key Request - DB.Get", "value", value)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
			logger.Info("key Request",
				"function", "key", "struct", "APIv1",
				"id", request.ID, "address", request.RequestIP,
				"user", request.GetUserName(), "method", request.Method,
				"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "key", request.Key, "status", status)
		*/
		status = http.StatusCreated
		App.WriteStatusMessage(status, w, request)
		return

	case "PUT":
		debugLogger.Debug("PUT Content", "value", request.Attachment.Value, "type", request.Attachment.Type)
		err := App.DB.Set(request.Namespace, request.Key, request.Attachment.Value)
		if err != nil {
			debugLogger.Debug("Error setting key in db", "Error", err)
			status := http.StatusInternalServerError
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		// Should not be nessesary to test that object is created....
		/*
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
		*/
		status = http.StatusCreated
		App.WriteStatusMessage(status, w, request)
		return
	case "UPDATE", "PATCH":
		status = http.StatusCreated
		debugLogger.Debug("UPDATE Content",
			"update.type", request.Attachment.Type,
			"update.value", request.Attachment.Value)
		newKey := request.Key
		if request.Key == "" { // Should never happen anymore :-/
			newKey = AuthGenerateRandomString(16)
		}
		newData := rest.KVPairV2{Key: newKey, Namespace: request.Namespace, Value: AuthGenerateRandomString(32)}

		_, err := App.DB.Get(request.Namespace, request.Key)
		exists := err == nil
		if !exists {
			if _, ok := err.(*ErrNotFound); !ok {
				status = http.StatusInternalServerError
				debugLogger.Debug("Error getting key in db", "Error", err)
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
				App.WriteStatusMessage(status, w, request)
				return
			}
		}
		debugLogger.Debug("UPDATE random",
			"newData.key", newData.Key, "newData.value", newData.Value, "exists", exists)
		if exists {
			if request.Attachment.Type == rest.TypeRoll {
				err := App.DB.Set(newData.Namespace, newData.Key, newData.Value)
				if err != nil {
					debugLogger.Debug("Error setting key in db", "Error", err)
					status = http.StatusInternalServerError
					keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
					App.WriteStatusMessage(status, w, request)
					return
				}
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
				request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", http.StatusText(status))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(newData)
				return
			}
		} else {
			if request.Attachment.Type == rest.TypeGenerate {
				err := App.DB.Set(newData.Namespace, newData.Key, newData.Value)
				if err != nil {
					debugLogger.Debug("Error setting key in db", "Error", err)
					status := http.StatusInternalServerError
					keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
					App.WriteStatusMessage(status, w, request)
					return
				}
				keys.WithLabelValues(request.Key, request.Namespace, request.Method, http.StatusText(status)).Inc()
				request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", http.StatusText(status))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(newData)
				return
			}
		}
		keys.WithLabelValues(request.Namespace, request.Key, "UPDATE", "BadRequest").Inc()
		App.WriteStatusMessage(http.StatusBadRequest, w, request)
		return
	case "DELETE":
		err := App.DB.DeleteKey(request.Namespace, request.Key)
		if err != nil {
			status := http.StatusInternalServerError
			if _, ok := err.(*ErrNotAllowed); ok {
				status = http.StatusForbidden
			}
			debugLogger.Debug("Error deleting key in db", "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	default:
		status = http.StatusNotFound
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	}
}

func (api *APIv1) namespace(w http.ResponseWriter, request *RequestParameters) {
	debugLogger := request.Logger.Ext.With("function", "namespace")
	status := http.StatusOK
	debugLogger.Debug("namespace Request - Start", "attachment", request.Attachment)
	switch request.Method {
	case "POST":
		if request.Attachment == nil {
			status = http.StatusBadRequest
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
		}
		namespace := request.Namespace
		if namespace == "" {
			namespace = request.Attachment.Value
		}
		if namespace == "" {
			status = http.StatusBadRequest
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
		}
		err := App.DB.CreateNamespace(namespace)
		if err != nil {
			status = http.StatusInternalServerError
			debugLogger.Debug("Error creating namespace in db", "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		status = http.StatusCreated
		keys.WithLabelValues(request.Key, namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	case "DELETE":
		err := App.DB.DeleteNamespace(request.Namespace)
		if err != nil {
			status = http.StatusInternalServerError
			if _, ok := err.(*ErrNotAllowed); ok {
				status = http.StatusForbidden
			}
			debugLogger.Debug("Error deleting namespace in db", "Error", err)
			keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
			App.WriteStatusMessage(status, w, request)
			return
		}
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
		return
	default:
		status = http.StatusNotFound
		keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
		App.WriteStatusMessage(status, w, request)
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
			if len(App.Config.PublicReadableNamespaces) > 0 {
				for _, namespace := range App.Config.PublicReadableNamespaces {
					if request.Namespace == namespace {
						return &ConfigPermissions{}
					}
				}
			}
			return &ConfigPermissions{Read: true}
		case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
			return &ConfigPermissions{Write: true}
		}
	}
	return &ConfigPermissions{Write: true, Read: true, List: true}
}
