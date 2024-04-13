package main

import (
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
	Auth       Auth
	Config     ConfigType
	Count      *Counter
	DB         Database
	HTTPServer *http.Server
	MTLSServer *http.Server
}

var decoder = schema.NewDecoder()

func (App *Application) RootControllerV1(w http.ResponseWriter, r *http.Request) {

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
			if strings.Contains(body, "key=") || strings.Contains(body, "value=") {
				return App.decodeXWWWForm(r, data)
			}
			construct := data.(*rest.KVPairV2)
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

func (App *Application) BadRequestHandler(w http.ResponseWriter, request *RequestParameters) {
	logger.Info("Auth Failed",
		"function", "ServeAuthFailed", "struct", "Auth",
		"id", request.ID, "address", request.RequestIP,
		"user", request.GetUserName(), "method", request.Method,
		"path", request.orgRequest.URL.EscapedPath(), "namespace", request.Namespace, "status", 400)
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 Bad Request"))
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (App *Application) ServeHTTP(mux *http.ServeMux) {
	App.HTTPServer = &http.Server{
		Addr:    ":" + App.Config.Port,
		Handler: mux,
	}
	log.Printf("I Serving on port %v", App.Config.Port)
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
		log.Printf("I Serving MTLS on port %v", App.Config.MTLS.Port)
		log.Fatal(App.MTLSServer.ListenAndServe())
	} else {
		if !checkFileExists(App.Config.MTLS.CACertificate) {
			log.Printf("E External MTLS not Enabled but no CACertificate exists: %v", App.Config.MTLS.CACertificate)
			missingFile = true
		}
		if !checkFileExists(App.Config.MTLS.Certificate) {
			log.Printf("E External MTLS not Enabled but no Certificate exists: %v", App.Config.MTLS.Certificate)
			missingFile = true
		}
		if !checkFileExists(App.Config.MTLS.Key) {
			log.Printf("E External MTLS not Enabled but no Key exists: %v", App.Config.MTLS.Key)
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
			log.Printf("I Serving MTLS on port %v", App.Config.MTLS.Port)
			log.Fatal(App.MTLSServer.ListenAndServeTLS(App.Config.MTLS.Certificate, App.Config.MTLS.Key))
		}
	}
}
