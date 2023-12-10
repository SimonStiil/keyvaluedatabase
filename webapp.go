package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/SimonStiil/keyvaluedatabase/rest"
	"github.com/gorilla/schema"
	"github.com/netinternet/remoteaddr"
)

type Application struct {
	Auth         Auth
	Config       ConfigType
	Count        *Counter
	DB           Database
	HTTPServer   *http.Server
	MTLSServer   *http.Server
	HostHeadders []string
}

func (App *Application) GreetingController(w http.ResponseWriter, r *http.Request) {
	requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	//https://stackoverflow.com/questions/64437991/prevent-http-handlefunc-funcw-r-handler-being-called-for-all-unmatc
	if !(r.URL.Path == "/system/greeting") {
		log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	log.Println("I Greetings-check")
	name := "World!"
	val := r.URL.Query()["name"]
	if len(val) > 0 {
		name = val[0]
	}
	reply := rest.GreetingV1{Id: App.Count.GetCount(), Content: "Hello, " + name}
	log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
	return
}

var decoder = schema.NewDecoder()

func (App *Application) RootController(w http.ResponseWriter, r *http.Request) {
	requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	//https://stackoverflow.com/questions/64437991/prevent-http-handlefunc-funcw-r-handler-being-called-for-all-unmatc
	slashSeperated := strings.Split(r.URL.Path[1:], "/")
	key := slashSeperated[0]
	method := r.Method
	slashes := strings.Count(r.URL.Path, "/")
	id := App.Count.GetCount()
	if App.Config.Debug {
		log.Printf("D %d RootController %v %v %v\n", id, method, key, slashes)
	}
	if len(slashSeperated) > 1 {
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404, "ToManySlashes")
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	switch method {
	case "GET":
		data := rest.KVPairV1{Key: key}
		if data.Key == "" {
			if !App.decodeAny(w, r, &data) {
				return
			}
		}
		if App.Config.Debug {
			log.Printf("D %d %v key: %v Value: %v\n", id, method, data.Key, data.Value)
		}
		if data.Key == "" {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		value, ok := App.DB.Get(key)
		if App.Config.Debug {
			log.Printf("D %d value(%v): %v\n", id, ok, value)
		}
		if !ok {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		reply := rest.KVPairV1{Key: key, Value: value}
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200, data.Key)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply)
		return
	case "POST":
		data := rest.KVPairV1{Key: key}
		if !App.decodeAny(w, r, &data) {
			return
		}
		if App.Config.Debug {
			log.Printf("D %d %v key: %v Value: %v\n", id, method, data.Key, data.Value)
		}
		if key != "" && key != data.Key {
			App.BadRequestHandler().ServeHTTP(w, r)
			return
		}
		App.DB.Set(data.Key, data.Value)
		value, ok := App.DB.Get(data.Key)
		if App.Config.Debug {
			log.Printf("D %d value(%v): %v\n", id, ok, value)
		}
		if !ok {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 201, data.Key)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
		return

	case "PUT":
		if key == "" {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		contenttype := r.Header.Get("Content-Type")
		var bodyBytes []byte
		var err error
		if r.Body != nil {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("Body reading error: %v", err)
				return
			}
			defer r.Body.Close()
		}
		if App.Config.Debug {
			log.Printf("D %d %v key: %v Content-Type: %v Length %v(%v)\n", id, method, key, contenttype, len(bodyBytes), r.ContentLength)
		}
		App.DB.Set(key, string(bodyBytes))
		value, ok := App.DB.Get(key)
		if App.Config.Debug {
			log.Printf("D %d value(%v): %v\n", id, ok, value)
		}
		if !ok {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 201, key)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
		return
	case "UPDATE", "PATCH":
		data := rest.KVUpdateV1{Key: key}
		var newData rest.KVPairV1
		if !App.decodeAny(w, r, &data) {
			return
		}
		if App.Config.Debug {
			log.Printf("D %d %v key: %v Type: %v\n", id, method, data.Key, data.Type)
		}
		if key != "" && key != data.Key {
			App.BadRequestHandler().ServeHTTP(w, r)
			return
		}
		if data.Key == "" {
			newData.Key = AuthGenerateRandomString(16)
		} else {
			newData.Key = data.Key
		}
		newData.Value = AuthGenerateRandomString(32)
		if App.Config.Debug {
			log.Printf("D %d value(%v): %v\n", id, newData.Key, newData.Value)
		}
		_, exists := App.DB.Get(data.Key)
		if data.Type == "roll" && exists {
			App.DB.Set(newData.Key, newData.Value)
			log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200, data.Key)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)

			if App.Config.Debug {
				log.Printf("D %d value roll\n", id)
			}
			return
		}
		if data.Type == "generate" && !exists {
			App.DB.Set(newData.Key, newData.Value)
			log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200, data.Key)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)
			if App.Config.Debug {
				log.Printf("D %d value generate\n", id)
			}
			return
		}
		App.BadRequestHandler().ServeHTTP(w, r)
		return
	case "DELETE":
		data := rest.KVPairV1{Key: key}
		if data.Key == "" {
			if !App.decodeAny(w, r, &data) {
				return
			}
		}
		if App.Config.Debug {
			log.Printf("D %d %v key: %v Value: %v\n", id, method, data.Key, data.Value)
		}
		App.DB.Delete(data.Key)
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 201, data.Key)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
		return
	default:
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

}

func (App *Application) decodeAny(w http.ResponseWriter, r *http.Request, data any) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" && r.ContentLength == 0 {
		return true
	}

	switch contentType {
	case "application/x-www-form-urlencoded":

		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				fmt.Printf("E Body reading error: %v", err)
				return false
			}
			defer r.Body.Close()
			body := string(bodyBytes)
			if strings.Contains(body, "key=") || strings.Contains(body, "key=") {
				return App.decodeXWWWForm(w, r, data)
			}
			construct := data.(*rest.KVPairV1)
			construct.Value = body
			return true
		}
	case "application/json":
		return App.decodeJson(w, r, data)
	}
	if App.Config.Debug {
		log.Printf("D Unknown Content-Type: %+v\n", contentType)
	}
	return false
}

func (App *Application) decodeJson(w http.ResponseWriter, r *http.Request, data any) bool {
	status := true
	defer func() {
		if rec := recover(); rec != nil {
			if App.Config.Debug {
				log.Printf("D Panic: %+v\n", rec)
			}
			App.BadRequestHandler().ServeHTTP(w, r)
			status = false
		}
	}()
	json.NewDecoder(r.Body).Decode(data)
	return status
}

func (App *Application) decodeXWWWForm(w http.ResponseWriter, r *http.Request, data any) bool {
	err := r.ParseForm()
	if err != nil {
		if App.Config.Debug {
			log.Printf("D ParseForm: %v, %t\n", err, err)
		}
		App.BadRequestHandler().ServeHTTP(w, r)
		return false
	}
	if App.Config.Debug {
		log.Printf("D ParseForm(PostForm): %v\n", r.PostForm)
	}
	err = decoder.Decode(data, r.PostForm)
	if err != nil {
		if App.Config.Debug {
			log.Printf("D ParseForm(Decode): %v, %v\n", err, err.Error())
		}
		App.BadRequestHandler().ServeHTTP(w, r)
		return false
	}
	return true
}

func (App *Application) ListController(w http.ResponseWriter, r *http.Request) {
	requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	//https://stackoverflow.com/questions/64437991/prevent-http-handlefunc-funcw-r-handler-being-called-for-all-unmatc
	if !(r.URL.Path == "/system/list") {
		log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	id := App.Count.GetCount()
	if App.Config.Debug {
		log.Printf("D %d ListController\n", id)
	}
	content := App.DB.Keys()
	log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(content)
	return
}

func (App *Application) FullListController(w http.ResponseWriter, r *http.Request) {
	requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	//https://stackoverflow.com/questions/64437991/prevent-http-handlefunc-funcw-r-handler-being-called-for-all-unmatc
	if !(r.URL.Path == "/system/fullList") {
		log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	id := App.Count.GetCount()
	if App.Config.Debug {
		log.Printf("D %d ListController\n", id)
	}
	content := App.DB.Keys()
	var fullList []rest.KVPairV1
	for _, key := range content {
		value, ok := App.DB.Get(key)
		if ok {
			fullList = append(fullList, rest.KVPairV1{Key: key, Value: value})
		} else {
			log.Printf("E Error reading key from db %v", key)
		}
	}
	log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fullList)
	return
}

func (App *Application) BadRequestHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("I %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 400)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 Bad Request"))
		return
	})
}

func (App *Application) HealthActuator(w http.ResponseWriter, r *http.Request) {
	if App.Config.Prometheus.Enabled {
		requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	}
	if !(r.URL.Path == "/system/health") {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	reply := rest.HealthV1{Status: "UP", Requests: int(App.Count.PeakCount())}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
	return
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// https://www.alexedwards.net/blog/basic-authentication-in-go
// https://medium.com/@matryer/the-http-handler-wrapper-technique-in-golang-updated-bc7fbcffa702
func (App *Application) BasicAuth(next http.HandlerFunc, permission *ConfigPermissions) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(App.Auth.Users) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if permission == nil {
			switch r.Method {
			case "GET":
				permission = &ConfigPermissions{Read: true}
				break
			case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
				permission = &ConfigPermissions{Write: true}
				break
			default:
				permission = &ConfigPermissions{Write: true, Read: true, List: true}
			}
		}
		if App.Config.Debug {
			log.Println("D BasicAuth for: ", GetFunctionName(next))
		}
		username, password, ok := r.BasicAuth()
		if ok {
			user, ok := App.Auth.Users[username]
			if ok {
				passwordHash := AuthHash(password)
				if AuthTest(passwordHash, user.PasswordEnc) {
					if AuthTestPermission(user.Permissions, *permission) {
						r.Header.Set("secret_remote_username", username)
						next.ServeHTTP(w, r)
						return
					}
				}
			}
		} else {
			username = "-"
		}
		log.Printf("I %v %v %v %v %v %v", r.Header.Get("secret_remote_address"), username, r.Method, r.URL.Path, 401, "BasicAuthCheckFailed")
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (App *Application) HostBlocker(next http.HandlerFunc, permission *ConfigPermissions) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(App.Config.Hosts) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		if permission == nil {
			switch r.Method {
			case "GET":
				permission = &ConfigPermissions{Read: true}
				break
			case "POST", "PUT", "UPDATE", "PATCH", "DELETE":
				permission = &ConfigPermissions{Write: true}
				break
			default:
				permission = &ConfigPermissions{Write: true, Read: true, List: true}
			}
		}
		var found bool
		denied := false
		address := r.Header.Get("secret_remote_address")
		foundHeadderName := r.Header.Get("secret_remote_header")
		for _, host := range App.Config.Hosts {
			if !strings.Contains(host.Address, "/") {
				if host.Address == address {
					found = true
					if AuthTestPermission(host.Permissions, *permission) {
						r.Header.Set("secret_remote_address", address)
						next.ServeHTTP(w, r)
						return
					} else {
						denied = true
						if App.Config.Debug {
							log.Println("D HostBlocker - ", foundHeadderName, " - AuthTestPermission failed for ", host.Address)
						}
						break
					}
				}
			}
		}
		if !denied {
			for _, host := range App.Config.Hosts {
				_, subnet, _ := net.ParseCIDR(host.Address)
				if subnet != nil {
					ip := net.ParseIP(address)
					if subnet.Contains(ip) {
						found = true
						if AuthTestPermission(host.Permissions, *permission) {
							r.Header.Set("secret_remote_address", address)
							next.ServeHTTP(w, r)
							return
						} else {
							if App.Config.Debug {
								log.Println("D HostBlocker(CIDR) - ", foundHeadderName, " - AuthTestPermission failed for ", host.Address)
							}
						}
					}

				}
			}
		}
		if !found && App.Config.Debug {
			log.Println("D HostBlocker - ", foundHeadderName, " - Lookup Host failed for ", address)
		}
		log.Printf("I %v %v %v %v %v %v", address, "-", r.Method, r.URL.Path, 401, "HostCheckFailed")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

func (App *Application) GetIP(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		address, _ := remoteaddr.Parse().IP(r)
		foundHeadderName := "remoteaddr"
		if App.Config.Debug {
			log.Println("D HostBlocker - ", "request from: ", address)
		}
		var trustedProxy bool
		for _, proxyAddress := range App.Config.TrustedProxies {
			if proxyAddress == address {
				if App.Config.Debug {
					log.Printf("D HostBlocker - trustedProxy - %v is trusted\n", address)
				}
				trustedProxy = true
				break
			}
		}
		if trustedProxy {
			for _, headderName := range App.HostHeadders {
				headder := r.Header[headderName]
				if len(headder) > 0 {
					if App.Config.Debug {
						log.Println("D HostBlocker - ", headderName, " - ", address)
					}
					address = headder[0]
					foundHeadderName = headderName
					break
				}
			}
		} else {
			if App.Config.Debug {
				log.Printf("D HostBlocker - trustedProxy - %v is not trusted, skipping headders\n", address)
			}
		}
		r.Header.Set("secret_remote_address", address)
		r.Header.Set("secret_remote_header", foundHeadderName)
		next.ServeHTTP(w, r)
	})
}

func (App *Application) SetMTLSUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("secret_remote_username", "mtls")
		next.ServeHTTP(w, r)
	})
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
