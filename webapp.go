package main

import (
	"bytes"
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
	debugLogger := request.Logger.Ext.With("function", "RootControllerV1")
	debugLogger.Debug("WebAppStart")
	request.RequestIP, _ = App.Auth.GetIPHeaderFromRequest(request)
	for _, api := range App.APIEndpoints {
		if api.APIPrefix() == request.Api {
			permissions := api.Permissions(request)
			debugLogger.Debug("Select Api", "prefix", api.APIPrefix(), "requiredPermissions", permissions)
			if permissions.globalAllowed() {
				debugLogger.Debug("Globally allowed API Request", "api", request.Api)
				api.ApiController(w, request)
			} else {
				if App.Auth.Authentication(request) && request.Authentication.User.Autorization(
					request,
					permissions) {
					debugLogger.Debug("Auth Successful", "prefix", api.APIPrefix(), "api", request.Api)
					api.ApiController(w, request)
					return
				} else {
					debugLogger.Debug("Auth Failed", "prefix", api.APIPrefix(), "api", request.Api)
					App.WriteStatusMessage(http.StatusUnauthorized, w, request)
				}
			}
		}
	}
}

func readAndResetBody(request *RequestParameters) error {
	r := request.orgRequest
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			return err
		}
		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		request.Body = string(bodyBytes)
	} else {
		return &ErrNotFound{Value: "body"}
	}
	return nil
}

func (App *Application) decodeAny(request *RequestParameters, data any) error {
	r := request.orgRequest
	contentType := r.Header.Get("Content-Type")
	debugLogger := request.Logger.Ext.With("function", "decodeAny", "contentType", contentType)

	if contentType == "" && r.ContentLength == 0 {
		return nil
	}
	err := readAndResetBody(request)
	if err != nil {
		debugLogger.Debug("ReadAll error", "error", err)
		return err
	}
	debugLogger.Debug("Decoding", "contentType", contentType, "body", request.Body)

	switch contentType {
	case "application/x-www-form-urlencoded":
		if r.Body != nil {
			if strings.Contains(request.Body, "type=") || strings.Contains(request.Body, "value=") {
				return App.decodeXWWWForm(request, data)
			}
			construct := data.(*rest.ObjectV1)
			construct.Value = strings.TrimSpace(request.Body)
			return nil
		}
	case "application/json":
		return App.decodeJson(request, data)
	case "":
		if request.Body != "" {
			construct := data.(*rest.ObjectV1)
			construct.Value = strings.TrimSpace(request.Body)
			return nil
		}
	}
	debugLogger.Debug("Unknown contenttype")
	return fmt.Errorf("unknown Content-Type: %v", contentType)
}

func (App *Application) decodeJson(request *RequestParameters, data any) error {
	r := request.orgRequest
	debugLogger := request.Logger.Ext.With("function", "decodeJson")
	var err error
	defer func() {
		if rec := recover(); rec != nil {
			debugLogger.Debug("json Decode Panic error", "error", rec)
			err = fmt.Errorf("%+v", rec)
		}
	}()
	json.NewDecoder(r.Body).Decode(data)
	construct := data.(*rest.ObjectV1)
	if construct.Type == "" && construct.Value == "" {
		return &ErrMalformRequest{Value: "Unable to parse Json Request"}
	}
	return err
}

func (App *Application) decodeXWWWForm(request *RequestParameters, data any) error {
	r := request.orgRequest
	debugLogger := request.Logger.Ext.With("function", "decodeXWWWForm")
	err := r.ParseForm()
	if err != nil {
		debugLogger.Debug("ParseForm error", "error", err)
		return err
	}
	debugLogger.Debug(fmt.Sprintf("ParseForm PostForm: %+v", r.PostForm))
	err = decoder.Decode(data, r.PostForm)
	if err != nil {
		debugLogger.Debug("Decode error", "error", err)
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
	request.Logger.Ext.Debug(statusTextFormated,
		"function", "WriteStatusMessage", "status", status, "status-text", statusText)
	request.Logger.Log.Info("Handeled Reqeust", "status", status, "status-text", statusText)
	w.Header().Set("Content-Type", "text/html")
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	}
	w.WriteHeader(status)
	w.Write([]byte(statusTextFormated))
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
