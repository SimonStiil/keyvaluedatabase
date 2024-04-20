package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
		Name: "http_endpoint_requests_count",
		Help: "The amount of requests to an endpoint",
	}, []string{"endpoint", "method"},
	)

	keys = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "key_request_count",
		Help: "The amount of requests for a certain key",
	}, []string{"key", "namespace", "method", "error"},
	)
	logger      *slog.Logger
	debugLogger *slog.Logger
	App         *Application
)

type ConfigType struct {
	Logging        ConfigLogging    `mapstructure:"logging"`
	Port           string           `mapstructure:"port"`
	DatabaseType   string           `mapstructure:"databaseType"`
	Users          []ConfigUser     `mapstructure:"users"`
	MTLS           MTLSConfig       `mapstructure:"mtls"`
	TrustedProxies []string         `mapstructure:"trustedProxies"`
	Redis          ConfigRedis      `mapstructure:"redis"`
	Mysql          ConfigMysql      `mapstructure:"mysql"`
	Prometheus     ConfigPrometheus `mapstructure:"prometheus"`
}
type ConfigLogging struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	File   string `mapstructure:"file"`
}

type ConfigUser struct {
	Username       string                 `mapstructure:"username"`
	Password       string                 `mapstructure:"password"`
	Permissionsset []ConfigPermissionsset `mapstructure:"permissionsset"`
	Hosts          []string               `mapstructure:"hosts"`
}

type ConfigPermissionsset struct {
	Namespaces  []string          `mapstructure:"namespaces"`
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
	configReader.SetDefault("logging.level", "Debug")
	configReader.SetDefault("logging.format", "text")
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
func setupLogging(Logging ConfigLogging) {
	logLevel := strings.ToLower(Logging.Level)
	logFormat := strings.ToLower(Logging.Format)
	loggingLevel := new(slog.LevelVar)
	switch logLevel {
	case "debug":
		loggingLevel.Set(slog.LevelDebug)
	case "warn":
		loggingLevel.Set(slog.LevelWarn)
	case "error":
		loggingLevel.Set(slog.LevelError)
	default:
		loggingLevel.Set(slog.LevelInfo)
	}
	switch logFormat {
	case "json":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel}))
		debugLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel, AddSource: true}))
	default:
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel}))
		debugLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel, AddSource: true}))
	}
	logger.Info("Logging started with options", "format", Logging.Format, "level", Logging.Level, "function", "setupLogging")
	slog.SetDefault(logger)
}

func setupTestlogging() {
	loggingLevel := new(slog.LevelVar)
	loggingLevel.Set(slog.LevelDebug)
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggingLevel, AddSource: true}))
	debugLogger = logger
}

// https://medium.com/mercadolibre-tech/go-language-relational-databases-and-orms-682a5fd3bbb6
func main() {
	flag.StringVar(&generate, "generate", "", "Generate an encrypted password to use for basic auth")
	flag.StringVar(&test, "test", "", "Test a base64hash versus a password")
	flag.StringVar(&configFileName, "config", "config", "Use a different config file name")
	flag.Parse()
	App = new(Application)
	ConfigRead(configFileName, &App.Config)
	// Logging setup
	setupLogging(App.Config.Logging)
	App.Auth.AuthGenerate(generate, test)
	//App.Auth.Init(App.Config)

	switch App.Config.DatabaseType {
	case "redis":
		logger.Info("Using Redis DB", "function", "main")
		App.DB = &RedisDatabase{Config: &App.Config.Redis}
		App.DB.Init()
	case "mysql":
		logger.Info("Using Maria DB", "function", "main")
		App.DB = &MariaDatabase{Config: &App.Config.Mysql}
		App.DB.Init()
	case "yaml":
		logger.Info("Using Yaml DB (no Redis or Mysql configuration)", "function", "main")
		App.DB = &YamlDatabase{}
		App.DB.Init()
	}
	App.Count = &Counter{}
	App.Count.Init(App.DB)
	App.Auth.Init(App.Config)
	App.APIEndpoints = []API{&Systemv1{}, &APIv1{}}
	defer App.DB.Close()
	if App.Config.Prometheus.Enabled {
		logger.Info(fmt.Sprintf("Metrics enabled at %v", App.Config.Prometheus.Endpoint), "function", "main")
		http.Handle(App.Config.Prometheus.Endpoint, promhttp.Handler())
	}
	regularServerMux := http.NewServeMux()
	regularServerMux.HandleFunc("/", http.HandlerFunc(App.RootControllerV1))

	logger.Info("users does not contain any entries, password auth disabled", "function", "main")
	if App.Config.MTLS.Enabled {
		mtlsServerMux := http.NewServeMux()
		mtlsServerMux.HandleFunc("/", http.HandlerFunc(App.RootControllerV1))
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
