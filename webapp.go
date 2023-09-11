package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/gorilla/schema"
	"github.com/netinternet/remoteaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	debug          bool
	generate       string
	test           string
	configFileName string
	port           string
	requests       = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_endpoint_equests_count",
		Help: "The amount of requests to an endpoint",
	}, []string{"endpoint", "method"},
	)
)

const (
	BaseENVname = "KVDB"
)

type Application struct {
	Auth     Auth
	Config   Config
	Handlers struct {
		GreetingController http.HandlerFunc
		HealthActuator     http.HandlerFunc
		RootController     http.HandlerFunc
		ListController     http.HandlerFunc
	}
	Count        Counter
	DB           Database
	HostHeadders []string
}

type Greeting struct {
	Id      uint32 `json:"id"`
	Content string `json:"content"`
}

type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type KVUpdate struct {
	Key  string `json:"key"`
	Type string `json:"type"`
}

func (App *Application) GreetingController(w http.ResponseWriter, r *http.Request) {
	requests.WithLabelValues(r.URL.EscapedPath(), r.Method).Inc()
	//https://stackoverflow.com/questions/64437991/prevent-http-handlefunc-funcw-r-handler-being-called-for-all-unmatc
	if !(r.URL.Path == "/system/greeting") {
		log.Printf("%v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	log.Println("Greetings-check")
	name := "World!"
	val := r.URL.Query()["name"]
	if len(val) > 0 {
		name = val[0]
	}
	reply := Greeting{App.Count.GetCount(), "Hello, " + name}
	log.Printf("%v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200)
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
	if debug {
		log.Printf("%d RootController %v %v %v\n", id, method, key, slashes)
	}
	if len(slashSeperated) > 1 {
		log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404, "ToManySlashes")
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	switch method {
	case "GET":
		data := KVPair{Key: key}
		if data.Key == "" {
			if !App.decodeAny(w, r, &data) {
				return
			}
		}
		if debug {
			log.Printf("%d %v key: %v Value: %v\n", id, method, data.Key, data.Value)
		}
		if data.Key == "" {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		value, ok := App.DB.Get(key)
		if debug {
			log.Printf("%d value(%v): %v\n", id, ok, value)
		}
		if !ok {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		reply := KVPair{Key: key, Value: value}
		log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200, data.Key)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reply)
		return
	case "POST", "PUT":
		data := KVPair{Key: key}
		if !App.decodeAny(w, r, &data) {
			return
		}
		if debug {
			log.Printf("%d %v key: %v Value: %v\n", id, method, data.Key, data.Value)
		}
		if key != "" && key != data.Key {
			App.BadRequestHandler().ServeHTTP(w, r)
			return
		}
		App.DB.Set(data.Key, data.Value)
		value, ok := App.DB.Get(data.Key)
		if debug {
			log.Printf("%d value(%v): %v\n", id, ok, value)
		}
		if !ok {
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 201, data.Key)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
		return
	case "UPDATE", "PATCH":
		data := KVUpdate{Key: key}
		var newData KVPair
		if !App.decodeAny(w, r, &data) {
			return
		}
		if debug {
			log.Printf("%d %v key: %v Type: %v\n", id, method, data.Key, data.Type)
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
		if debug {
			log.Printf("%d value(%v): %v\n", id, newData.Key, newData.Value)
		}
		_, exists := App.DB.Get(data.Key)
		if data.Type == "roll" && exists {
			App.DB.Set(data.Key, newData.Value)
			log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200, data.Key)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)

			if debug {
				log.Printf("%d value roll\n", id)
			}
			return
		}
		if data.Type == "generate" && !exists {
			App.DB.Set(data.Key, newData.Value)
			log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200, data.Key)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(newData)
			if debug {
				log.Printf("%d value generate\n", id)
			}
			return
		}
		App.BadRequestHandler().ServeHTTP(w, r)
		return
	case "DELETE":
		data := KVPair{Key: key}
		if data.Key == "" {
			if !App.decodeAny(w, r, &data) {
				return
			}
		}
		if debug {
			log.Printf("%d %v key: %v Value: %v\n", id, method, data.Key, data.Value)
		}
		App.DB.Delete(data.Key)
		log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 201, data.Key)
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
		return App.decodeXWWWForm(w, r, data)
	case "application/json":
		return App.decodeJson(w, r, data)
	}
	if debug {
		log.Printf("Unknown Content-Type: %+v\n", contentType)
	}
	return false
}

func (App *Application) decodeJson(w http.ResponseWriter, r *http.Request, data any) bool {
	status := true
	defer func() {
		if rec := recover(); rec != nil {
			if debug {
				log.Printf("Panic: %+v\n", rec)
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
		if debug {
			log.Printf("ParseForm: %v\n", err)
		}
		App.BadRequestHandler().ServeHTTP(w, r)
		return false
	}
	err = decoder.Decode(data, r.PostForm)
	if err != nil {
		if debug {
			log.Printf("ParseForm: %v\n", err)
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
		log.Printf("%v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 404)
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	id := App.Count.GetCount()
	if debug {
		log.Printf("%d ListController\n", id)
	}
	content := App.DB.Keys()
	log.Printf("%v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 200)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(content)
	return
}

type Health struct {
	Status   string `json:"status"`
	Requests int    `json:"requests"`
}

func (App *Application) BadRequestHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v %v %v %v %v", r.Header.Get("secret_remote_address"), r.Header.Get("secret_remote_username"), r.Method, r.URL.Path, 400)
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
	reply := Health{Status: "UP", Requests: int(App.Count.PeakCount())}
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
		if debug {
			log.Println("BasicAuth for: ", GetFunctionName(next))
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
		log.Printf("%v %v %v %v %v %v", r.Header.Get("secret_remote_address"), username, r.Method, r.URL.Path, 401, "BasicAuthCheckFailed")
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
		ip, _ := remoteaddr.Parse().IP(r)
		address := ip
		foundHeadderName := "remoteaddr"
		if debug {
			log.Println("HostBlocker - ", "request from: ", address)
		}
		var trustedProxy bool
		for _, proxyAddress := range App.Config.TrustedProxies {
			if proxyAddress == address {
				if debug {
					log.Printf("HostBlocker - trustedProxy - %v is trusted\n", address)
				}
				trustedProxy = true
				break
			}
		}
		if trustedProxy {
			for _, headderName := range App.HostHeadders {
				headder := r.Header[headderName]
				if len(headder) > 0 {
					if debug {
						log.Println("HostBlocker - ", headderName, " - ", address)
					}
					address = headder[0]
					foundHeadderName = headderName
					break
				}
			}
		} else {
			if debug {
				log.Printf("HostBlocker - trustedProxy - %v is not trusted, skipping headders\n", address)
			}
		}

		var found bool
		for _, host := range App.Config.Hosts {
			if host.Address == address {
				found = true
				if AuthTestPermission(host.Permissions, *permission) {
					r.Header.Set("secret_remote_address", address)
					next.ServeHTTP(w, r)
					return
				} else {
					if debug {
						log.Println("HostBlocker - ", foundHeadderName, " - AuthTestPermission failed for ", host.Address)
					}
				}
			}
		}
		if !found && debug {
			log.Println("HostBlocker - ", foundHeadderName, " - Lookup Host failed for ", address)
		}
		log.Printf("%v %v %v %v %v %v", address, "-", r.Method, r.URL.Path, 401, "HostCheckFailed")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// https://medium.com/mercadolibre-tech/go-language-relational-databases-and-orms-682a5fd3bbb6
func main() {
	flag.BoolVar(&debug, "debug", false, "Enable Debugging output")

	flag.StringVar(&generate, "generate", "", "Generate an encrypted password to use for basic auth")
	flag.StringVar(&test, "test", "", "Test a base64hash versus a password")
	flag.StringVar(&configFileName, "config", "config.yaml", "Use a different config file name")
	flag.StringVar(&port, "port", "8080", "port to use for the service")
	flag.Parse()
	debugString := strings.ToLower(os.Getenv(BaseENVname + "_DEBUG"))
	if !debug {
		debug = (debugString != "" && debugString != "false" && debugString != "no")
	}
	if debug {
		log.Println("Debugging: ", debug)
	}
	AuthGenerate(generate, test)

	log.Println("Starting")
	App := new(Application)
	App.Config = ConfigRead(configFileName)
	App.Auth.Init(App.Config)

	App.HostHeadders = []string{
		"X-Forwarded-For",
		"HTTP_FORWARDED",
		"HTTP_FORWARDED_FOR",
		"HTTP_X_FORWARDED",
		"HTTP_X_FORWARDED_FOR",
		"HTTP_CLIENT_IP",
		"HTTP_VIA",
		"HTTP_X_CLUSTER_CLIENT_IP",
		"Proxy-Client-IP",
		"WL-Proxy-Client-IP",
		"REMOTE_ADDR"}

	if App.Config.Redis.Address == "" {
		log.Printf("Using Yaml DB. (%v_REDIS_HOST missing)\n", BaseENVname)
		App.DB = new(YamlDatabase)
		App.DB.Init("", "")
	} else {
		log.Printf("Using Redis DB\n")
		App.DB = new(RedisDatabase)
		App.DB.Init(App.Config.Redis.Address, App.Config.Redis.Password)
	}
	App.Count.Init(App.DB)
	defer App.DB.Close()
	App.Handlers.GreetingController = http.HandlerFunc(App.GreetingController)
	App.Handlers.RootController = http.HandlerFunc(App.RootController)
	App.Handlers.ListController = http.HandlerFunc(App.ListController)
	App.Handlers.HealthActuator = http.HandlerFunc(App.HealthActuator)
	if App.Config.Prometheus.Enabled {
		endpoint := "/system/metrics"
		if App.Config.Prometheus.Endpoint != "" {
			endpoint = App.Config.Prometheus.Endpoint
		}
		log.Printf("Metrics enabled at %v\n", endpoint)
		http.Handle(endpoint, promhttp.Handler())
	}
	ListPermission := &ConfigPermissions{List: true}
	http.HandleFunc("/system/greeting", App.HostBlocker(App.BasicAuth(App.Handlers.GreetingController, nil), nil))
	http.HandleFunc("/", App.HostBlocker(App.BasicAuth(App.Handlers.RootController, nil), nil))
	http.HandleFunc("/system/list", App.HostBlocker(App.BasicAuth(App.Handlers.ListController, ListPermission), ListPermission))

	http.HandleFunc("/system/health", App.Handlers.HealthActuator)
	if debug {
		if len(App.Config.Hosts) == 0 {
			log.Println("config: hosts does not contain any entries, all hosts allowed")
		}
		if len(App.Auth.Users) == 0 {
			log.Println("config: users does not contain any entries, password auth disabled")
		}
	}
	log.Println("Serving on port 8080")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
