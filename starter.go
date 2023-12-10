package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

var (
	generate       string
	test           string
	configFileName string
	requests       = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_endpoint_equests_count",
		Help: "The amount of requests to an endpoint",
	}, []string{"endpoint", "method"},
	)
)

type ConfigType struct {
	Debug          bool             `mapstructure:"debug"`
	Port           string           `mapstructure:"port"`
	DatabaseType   string           `mapstructure:"databaseType"`
	Users          []ConfigUser     `mapstructure:"users"`
	Hosts          []ConfigHosts    `mapstructure:"hosts"`
	MTLS           MTLSConfig       `mapstructure:"mtls"`
	TrustedProxies []string         `mapstructure:"trustedProxies"`
	Redis          ConfigRedis      `mapstructure:"redis"`
	Mysql          ConfigMysql      `mapstructure:"mysql"`
	Prometheus     ConfigPrometheus `mapstructure:"prometheus"`
}

type ConfigUser struct {
	Username    string            `mapstructure:"username"`
	Password    string            `mapstructure:"password"`
	Permissions ConfigPermissions `mapstructure:"permissions"`
}

type ConfigHosts struct {
	Address     string            `mapstructure:"address"`
	Permissions ConfigPermissions `mapstructure:"permissions"`
}

type ConfigPermissions struct {
	Read  bool `mapstructure:"read"`
	Write bool `mapstructure:"write"`
	List  bool `mapstructure:"list"`
}
type ConfigPrometheus struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

type MTLSConfig struct {
	Enabled       bool              `mapstructure:"enabled"`
	Port          string            `mapstructure:"port"`
	Certificate   string            `mapstructure:"certificate"`
	CACertificate string            `mapstructure:"caCertificate"`
	Key           string            `mapstructure:"key"`
	ExternalMTLS  bool              `mapstructure:"externalMTLS"`
	Permissions   ConfigPermissions `mapstructure:"permissions"`
}

const (
	BaseENVname = "KVDB"
)

func ConfigRead(configFileName string, configOutput *ConfigType) {
	configReader := viper.New()
	configReader.SetConfigName(configFileName)
	configReader.SetConfigType("yaml")
	configReader.AddConfigPath("/app/")
	configReader.AddConfigPath(".")
	configReader.SetEnvPrefix(BaseENVname)
	MariaDBGetDefaults(configReader)
	RedisDBGetDefaults(configReader)
	configReader.SetDefault("debug", false)
	configReader.SetDefault("port", 8080)
	configReader.SetDefault("databaseType", "yaml")
	configReader.SetDefault("prometheus.enabled", true)
	configReader.SetDefault("prometheus.endpoint", "/system/metrics")
	configReader.SetDefault("mtls.enabled", false)
	configReader.SetDefault("mtls.port", 8443)
	configReader.SetDefault("mtls.certificate", "server.crt")
	configReader.SetDefault("mtls.key", "server.key")
	configReader.SetDefault("mtls.caCertificate", "ca.crt")
	configReader.SetDefault("mtls.externalMTLS", false)
	configReader.SetDefault("mtls.permissions.read", true)
	configReader.SetDefault("mtls.permissions.write", false)
	configReader.SetDefault("mtls.permissions.list", true)

	err := configReader.ReadInConfig() // Find and read the config file
	if err != nil {                    // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	configReader.AutomaticEnv()
	configReader.Unmarshal(configOutput)
}

// https://medium.com/mercadolibre-tech/go-language-relational-databases-and-orms-682a5fd3bbb6
func main() {
	flag.StringVar(&generate, "generate", "", "Generate an encrypted password to use for basic auth")
	flag.StringVar(&test, "test", "", "Test a base64hash versus a password")
	flag.StringVar(&configFileName, "config", "config", "Use a different config file name")
	flag.Parse()
	App := new(Application)
	log.Println("I Reading Configuration")
	ConfigRead(configFileName, &App.Config)

	if App.Config.Debug {
		log.Println("D Debugging: ", App.Config.Debug)
	}
	App.Auth.AuthGenerate(generate, test)
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
	switch App.Config.DatabaseType {
	case "redis":
		log.Printf("I Using Redis DB\n")
		App.DB = &RedisDatabase{Config: &App.Config}
		App.DB.Init()
	case "mysql":
		log.Printf("I Using Maria DB\n")
		App.DB = &MariaDatabase{Config: &App.Config}
		App.DB.Init()
	case "yaml":
		log.Print("I Using Yaml DB. (no Redis or Mysql configuration)\n")
		App.DB = &YamlDatabase{Config: &App.Config}
		App.DB.Init()
	}
	App.Count = &Counter{Config: &App.Config}
	App.Count.Init(App.DB)
	defer App.DB.Close()
	greetingController := http.HandlerFunc(App.GreetingController)
	rootController := http.HandlerFunc(App.RootController)
	listController := http.HandlerFunc(App.ListController)
	fullListController := http.HandlerFunc(App.FullListController)
	healthActuator := http.HandlerFunc(App.HealthActuator)
	if App.Config.Prometheus.Enabled {
		log.Printf("I Metrics enabled at %v\n", App.Config.Prometheus.Endpoint)
		http.Handle(App.Config.Prometheus.Endpoint, promhttp.Handler())
	}
	ListPermission := &ConfigPermissions{List: true}
	FullListPermission := &ConfigPermissions{List: true, Read: true}
	regularServerMux := http.NewServeMux()
	regularServerMux.HandleFunc("/system/greeting", App.GetIP(App.HostBlocker(App.BasicAuth(greetingController, nil), nil)))
	regularServerMux.HandleFunc("/", App.GetIP(App.HostBlocker(App.BasicAuth(rootController, nil), nil)))
	regularServerMux.HandleFunc("/system/list", App.GetIP(App.HostBlocker(App.BasicAuth(listController, ListPermission), ListPermission)))
	regularServerMux.HandleFunc("/system/fullList", App.GetIP(App.HostBlocker(App.BasicAuth(fullListController, FullListPermission), FullListPermission)))
	regularServerMux.HandleFunc("/system/health", healthActuator)
	if App.Config.Debug {
		if len(App.Config.Hosts) == 0 {
			log.Println("D config: hosts does not contain any entries, all hosts allowed")
		}
		if len(App.Auth.Users) == 0 {
			log.Println("D config: users does not contain any entries, password auth disabled")
		}
	}
	if App.Config.MTLS.Enabled {
		mtlsServerMux := http.NewServeMux()
		mtlsServerMux.HandleFunc("/system/greeting", App.SetMTLSUser(App.GetIP(greetingController)))
		mtlsServerMux.HandleFunc("/", App.SetMTLSUser(App.GetIP(rootController)))
		mtlsServerMux.HandleFunc("/system/list", App.SetMTLSUser(App.GetIP(listController)))
		mtlsServerMux.HandleFunc("/system/fullList", App.SetMTLSUser(App.GetIP(fullListController)))
		mtlsServerMux.HandleFunc("/system/health", App.SetMTLSUser(App.GetIP(healthActuator)))
		go App.ServeHTTP(regularServerMux)
		go App.ServeHTTPMTLS(mtlsServerMux)
		sigInterruptChannel := make(chan os.Signal, 1)
		signal.Notify(sigInterruptChannel, os.Interrupt)
		// block execution from continuing further until SIGINT comes
		<-sigInterruptChannel

		// create a context which will expire after 4 seconds of grace period
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*4)
		defer cancel()

		// ask to shutdown for both servers
		go App.HTTPServer.Shutdown(ctx)
		go App.MTLSServer.Shutdown(ctx)

		// wait until ctx ends (which will happen after 4 seconds)
		<-ctx.Done()
	} else {
		App.ServeHTTP(regularServerMux)
	}
}
