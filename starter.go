package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
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
	logFile     *os.File
	App         *Application
)

type ConfigType struct {
	Logging                  ConfigLogging    `mapstructure:"logging"`
	Port                     string           `mapstructure:"port"`
	DatabaseType             string           `mapstructure:"databaseType"`
	Users                    []ConfigUser     `mapstructure:"users"`
	OIDC                     ConfigOIDC       `mapstructure:"oidc"`
	TrustedProxies           []string         `mapstructure:"trustedProxies"`
	PublicReadableNamespaces []string         `mapstructure:"publicReadableNamespaces"`
	Redis                    ConfigRedis      `mapstructure:"redis"`
	Mysql                    ConfigMysql      `mapstructure:"mysql"`
	Postgres                 ConfigPostgres   `mapstructure:"postgres"`
	Prometheus               ConfigPrometheus `mapstructure:"prometheus"`
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

func (perm *ConfigPermissions) globalAllowed() bool {
	return !perm.List && !perm.Read && !perm.Write
}

type ConfigPrometheus struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

type ConfigOIDC struct {
	Enabled          bool     `mapstructure:"enabled"`
	ProviderURL      string   `mapstructure:"providerURL"`
	ClientID         string   `mapstructure:"clientID"`
	ClientSecret     string   `mapstructure:"clientSecret"`
	EnvVariableName  string   `mapstructure:"envVariableName"`
	RedirectURL      string   `mapstructure:"redirectURL"`
	Scopes           []string `mapstructure:"scopes"`
	TokenTTL         int      `mapstructure:"tokenTTL"`
	DisableBasicAuth bool     `mapstructure:"disableBasicAuth"`
	CookieName       string   `mapstructure:"cookieName"`
	CookieDomain     string   `mapstructure:"cookieDomain"`
}

const (
	BaseENVname = "KVDB"
)

func ConfigRead(configFileName string, configOutput *ConfigType) *viper.Viper {
	configReader := viper.New()
	configReader.SetConfigName(configFileName)
	configReader.SetConfigType("yaml")
	configReader.AddConfigPath("/app/")
	configReader.AddConfigPath(".")
	configReader.SetEnvPrefix(BaseENVname)
	MariaDBGetDefaults(configReader)
	RedisDBGetDefaults(configReader)
	PostgresGetDefaults(configReader)
	configReader.SetDefault("logging.level", "Debug")
	configReader.SetDefault("logging.format", "text")
	configReader.SetDefault("port", 8080)
	configReader.SetDefault("databaseType", "yaml")
	configReader.SetDefault("prometheus.enabled", true)
	configReader.SetDefault("prometheus.endpoint", "/system/metrics")
	configReader.SetDefault("oidc.enabled", false)
	configReader.SetDefault("oidc.providerURL", "http://127.0.0.1:9096/.well-known/openid-configuration")
	configReader.SetDefault("oidc.redirectURL", "http://127.0.0.1:8080/oidc/callback")
	configReader.SetDefault("oidc.scopes", []string{"openid", "profile", "email"})
	configReader.SetDefault("oidc.tokenTTL", 60)
	configReader.SetDefault("oidc.disableBasicAuth", false)
	configReader.SetDefault("oidc.cookieName", "kvdb_oidc_session")

	err := configReader.ReadInConfig() // Find and read the config file
	if err != nil {                    // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	configReader.AutomaticEnv()
	configReader.Unmarshal(configOutput)
	return configReader
}

func SetupConfigWatcher(logger *slog.Logger, configReader *viper.Viper, App *Application) {
	configReader.OnConfigChange(func(e fsnotify.Event) {
		logger.Info(fmt.Sprintf("Config file changed: %v", e.Name))
		App.Auth.LoadConfig(App.Config)
	})
	configReader.WatchConfig()
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
	if logFile != nil {
		defer func(f io.Closer) { _ = f.Close() }(logFile)
	}
	output := os.Stdout
	if Logging.File != "" {
		var err error
		logFile, err = os.OpenFile(Logging.File, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			log.Printf("Error opening log file for writing %v : %v", Logging.File, err)

		} else {
			output = logFile
		}

	}
	switch logFormat {
	case "json":
		logger = slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: loggingLevel}))
		debugLogger = slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: loggingLevel, AddSource: true}))
	default:
		logger = slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: loggingLevel}))
		debugLogger = slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: loggingLevel, AddSource: true}))
	}
	logger.Info("Logging started with options", "format", Logging.Format, "level", Logging.Level, "function", "setupLogging")
	//slog.SetDefault(logger)
}

var rotateSig = make(chan os.Signal, 1)

func logRotateHandler() {
	for {
		sig := <-rotateSig
		if sig == syscall.SIGHUP {
			logger.Info(fmt.Sprintf("Closing and re-opening log files for rotation: %+v", sig))
			setupLogging(App.Config.Logging)
		}
	}
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
	signal.Notify(rotateSig, syscall.SIGHUP)
	go logRotateHandler()
	App = new(Application)
	configReader := ConfigRead(configFileName, &App.Config)
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
	case "postgres":
		logger.Info("Using Postgres DB", "function", "main")
		App.DB = &PostgresDatabase{Config: &App.Config.Postgres}
		App.DB.Init()
	case "yaml":
		logger.Info("Using Yaml DB (no Redis or Mysql configuration)", "function", "main")
		App.DB = &YamlDatabase{}
		App.DB.Init()
	}
	App.Count = &Counter{}
	App.Count.Init(App.DB)
	App.Auth.Init(App.Config)
	SetupConfigWatcher(logger, configReader, App)
	App.APIEndpoints = []API{&Systemv1{}, &APIv1{}}
	defer App.DB.Close()
	if App.Config.Prometheus.Enabled {
		logger.Info(fmt.Sprintf("Metrics enabled at %v", App.Config.Prometheus.Endpoint), "function", "main")
		http.Handle(App.Config.Prometheus.Endpoint, promhttp.Handler())
	}
	regularServerMux := http.NewServeMux()
	regularServerMux.HandleFunc("/", http.HandlerFunc(App.RootControllerV1))

	if App.Config.OIDC.Enabled {
		logger.Info("OIDC enabled, registering OIDC endpoints", "function", "main")
		regularServerMux.HandleFunc("/oidc/login", http.HandlerFunc(App.Auth.OIDCLogin))
		regularServerMux.HandleFunc("/oidc/callback", http.HandlerFunc(App.Auth.OIDCCallback))
		regularServerMux.HandleFunc("/oidc/logout", http.HandlerFunc(App.Auth.OIDCLogout))
	}
	App.ServeHTTP(regularServerMux)
}
