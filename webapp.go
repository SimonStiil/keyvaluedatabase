package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	"github.com/gorilla/schema"
)

type Application struct {
	Auth         Auth
	Config       ConfigType
	Count        *Counter
	DB           Database
	HTTPServer   *http.Server
	MTLSServer   *http.Server
	APIEndpoints []API
}

var decoder = schema.NewDecoder()

func (App *Application) RootControllerV1(w http.ResponseWriter, r *http.Request) {
	requestcount := App.Count.GetCount()
	request := GetRequestParameters(r, requestcount)
	logger.Debug("Start",
		"function", "RootControllerV1", "struct", "Application",
		"id", request.ID, "address", request.RequestIP, "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace)
	for _, api := range App.APIEndpoints {
		if api.APIPrefix() == request.Api {

			logger.Debug("Select Api",
				"function", "RootControllerV1", "struct", "Application",
				"id", request.ID, "prefix", api.APIPrefix(),
				"api", request.Api)

			if App.Auth.Authentication(request) && request.Authentication.User.Autorization(
				request,
				api.Permissions(request)) {
				logger.Debug("Auth Successful",
					"function", "RootControllerV1", "struct", "Application",
					"id", request.ID, "prefix", api.APIPrefix(),
					"api", request.Api, "username", request.Basic.Username, "namespace", request.Namespace)
				api.ApiController(w, request)
				return
			} else {
				logger.Debug("Auth Failed",
					"function", "RootControllerV1", "struct", "Application",
					"id", request.ID, "prefix", api.APIPrefix(),
					"api", request.Api, "username", request.Basic.Username, "namespace", request.Namespace)
				App.Auth.ServeAuthFailed(w, request)
			}

		}
	}
}

func (App *Application) decodeAny(r *http.Request, data any) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" && r.ContentLength == 0 {
		return nil
	}

	switch contentType {
	case "application/x-www-form-urlencoded":

		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Debug("ReadAll error", "function", "decodeAny", "struct", "Application", "contentType", contentType, "error", err)
				return err
			}
			defer r.Body.Close()
			body := string(bodyBytes)
			if strings.Contains(body, "type=") || strings.Contains(body, "value=") {
				return App.decodeXWWWForm(r, data)
			}
			construct := data.(*rest.ObjectV1)
			construct.Value = body
			return nil
		}
	case "application/json":
		return App.decodeJson(r, data)
	}
	logger.Debug("Unknown contenttype", "function", "decodeAny", "struct", "Application", "contentType", contentType)
	return fmt.Errorf("unknown Content-Type: %v", contentType)
}

func (App *Application) decodeJson(r *http.Request, data any) error {
	var err error
	defer func() {
		if rec := recover(); rec != nil {
			logger.Debug("json Decode Panic error", "function", "decodeXWWWForm", "struct", "Application", "error", rec)
			err = fmt.Errorf("%+v", rec)
		}
	}()
	json.NewDecoder(r.Body).Decode(data)
	return err
}

func (App *Application) decodeXWWWForm(r *http.Request, data any) error {
	err := r.ParseForm()
	if err != nil {
		logger.Debug("ParseForm error", "function", "decodeXWWWForm", "struct", "Application", "error", err)
		return err
	}
	logger.Debug(fmt.Sprintf("ParseForm PostForm: %+v", r.PostForm), "function", "decodeXWWWForm", "struct", "Application")
	err = decoder.Decode(data, r.PostForm)
	if err != nil {
		logger.Debug("Decode error", "function", "decodeXWWWForm", "struct", "Application", "error", err)
		return err
	}
	return nil
}

func (App *Application) PrometheusStatusTest(status int) string {
	return strings.ReplaceAll(http.StatusText(status), " ", "")
}

func (App *Application) WriteStatusMessage(status int, w http.ResponseWriter, request *RequestParameters) {
	statusText := http.StatusText(status)
	statusTextFormated := fmt.Sprintf("%v %v", status, statusText)
	logger.Debug(statusTextFormated,
		"function", "WriteStatusMessage", "struct", "Auth",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "status", status, "status-text", statusText)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write([]byte(statusTextFormated))
}

func (App *Application) HTTPErrorHandler(logger *slog.Logger, err error, w http.ResponseWriter, request *RequestParameters) {
	status := http.StatusInternalServerError
	logger.Debug("HTTP Error", "Error", err)
	if _, ok := err.(*ErrNotFound); ok {
		status = http.StatusNotFound

	}
	keys.WithLabelValues(request.Key, request.Namespace, request.Method, App.PrometheusStatusTest(status)).Inc()
	App.WriteStatusMessage(status, w, request)
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (App *Application) ServeHTTP(mux *http.ServeMux) {
	App.HTTPServer = &http.Server{
		Addr:    ":" + App.Config.Port,
		Handler: mux,
	}
	logger.Info(fmt.Sprintf("Serving on port %v", App.Config.Port))
	log.Fatal(App.HTTPServer.ListenAndServe())
}
func checkFileExists(filePath string) bool {
	_, error := os.Stat(filePath)
	return !errors.Is(error, os.ErrNotExist)
}

func (App *Application) ServeHTTPMTLS(mux *http.ServeMux) {
	missingFile := false
	if App.Config.MTLS.ExternalMTLS {
		App.MTLSServer = &http.Server{
			Addr:    ":" + App.Config.MTLS.Port,
			Handler: mux,
		}
		logger.Info(fmt.Sprintf("Serving MTLS on port %v", App.Config.MTLS.Port))
		log.Fatal(App.MTLSServer.ListenAndServe())
	} else {
		if !checkFileExists(App.Config.MTLS.CACertificate) {
			logger.Error(fmt.Sprintf("External MTLS not Enabled but no CACertificate exists: %v", App.Config.MTLS.CACertificate))

			missingFile = true
		}
		if !checkFileExists(App.Config.MTLS.Certificate) {
			logger.Error(fmt.Sprintf("External MTLS not Enabled but no Certificate exists: %v", App.Config.MTLS.Certificate))
			missingFile = true
		}
		if !checkFileExists(App.Config.MTLS.Key) {
			logger.Error(fmt.Sprintf("External MTLS not Enabled but no Key exists: %v", App.Config.MTLS.Key))
			missingFile = true
		}
		if !missingFile {
			caCert, err := os.ReadFile(App.Config.MTLS.CACertificate)
			if err != nil {
				log.Fatal(err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig := &tls.Config{
				ClientCAs:  caCertPool,
				ClientAuth: tls.RequireAndVerifyClientCert,
			}
			App.MTLSServer = &http.Server{
				Addr:      ":" + App.Config.MTLS.Port,
				TLSConfig: tlsConfig,
				Handler:   mux,
			}
			logger.Info(fmt.Sprintf("Serving MTLS on port %v", App.Config.MTLS.Port))
			log.Fatal(App.MTLSServer.ListenAndServeTLS(App.Config.MTLS.Certificate, App.Config.MTLS.Key))
		}
	}
}
