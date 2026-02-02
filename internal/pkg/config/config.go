package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host    string   `yaml:"host"`
	Port    string   `yaml:"port"`
	Timeout int      `yaml:"timeout"`
	Origins []string `yaml:"origins"`
	Headers []string `yaml:"headers"`
	Methods []string `yaml:"methods"`
}

type MongoConfig struct {
	Username string `env:"MONGO_INITDB_ROOT_USERNAME"`
	Password string `env:"MONGO_INITDB_ROOT_PASSWORD"`
	Host     string `env:"MONGO_HOST"`
	Port     string `env:"MONGO_PORT"`
	DBName   string `env:"MONGO_DB"`
}

type PostgresConfig struct {
	Username string `env:"POSTGRES_USERNAME"`
	Password string `env:"POSTGRES_PASSWORD"`
	Host     string `env:"POSTGRES_HOST"`
	Port     string `env:"POSTGRES_PORT"`
	DBName   string `env:"POSTGRES_DB"`
}

type RedisConfig struct {
	Password string `env:"REDIS_PASSWORD"`
	Host     string `env:"REDIS_HOST"`
	Port     string `env:"REDIS_PORT"`
	DB       int    `env:"REDIS_DB"`
}

type MinioConfig struct {
	Host             string `env:"MINIO_HOST"`
	EndpointUser     string `env:"MINIO_ROOT_USER"`
	EndpointPassword string `env:"MINIO_ROOT_PASSWORD"`
	Port             string `env:"MINIO_API_PORT"`

	AccessKey string `env:"S3_ACCESS_KEY"`
	SecretKey string `env:"S3_SECRET_KEY"`
}

type ServiceConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Socks5ProxieConfig struct {
	IP   string `env:"SOCKS5_PROXIE_IP"`
	Port string `env:"SOCKS5_PROXIE_PORT"`
}

type VKMethodConfig struct {
	URL         string `yaml:"url"`
	Method      string `yaml:"method"`
	ContentType string `yaml:"content_type"`
}

type VKApiConfig struct {
	RedirectURI string         `env:"REDIRECT_URI"`
	ClientID    string         `env:"CLIENT_ID"`
	SecretKey   string         `env:"SECRET_KEY"`
	ServiceKey  string         `env:"SERVICE_KEY"`
	Exchange    VKMethodConfig `yaml:"exchange"`
	PublicInfo  VKMethodConfig `yaml:"public_info"`
}

type GoogleOAuthConfig struct {
	ClientID     string `env:"GOOGLE_CLIENT_ID"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	RedirectURI  string `env:"GOOGLE_REDIRECT_URI"`
}

type GeminiConfig struct {
	Host        string `env:"GEMINI_HOST"`
	Port        string `env:"GEMINI_PORT"`
	ExternalVM1 string `env:"EXTERNAL_VM_1_API_KEY"`
}

type ServicesConfig struct {
	Description     ServiceConfig `yaml:"description"`
	ActionProcessor ServiceConfig `yaml:"action_processor"`
}

type ProxiesConfig struct {
	Socks5Proxie Socks5ProxieConfig
}

type SessionConfig struct {
	Duration time.Duration `yaml:"duration" env:"SESSION_DURATION" env-default:"720h"`
}

type LoggerConfig struct {
	// Deprecated: Key is no longer used. The logger context key is now a typed
	// struct (logger.loggerCtxKey) and does not need external configuration.
	// This field will be removed in a future version.
	Key        string `yaml:"key"`
	OutputPath string `yaml:"path"`
	ErrPath    string `yaml:"err_path"`
}

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Session SessionConfig `yaml:"session"`

	Mongo    MongoConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Minio    MinioConfig

	Services    ServicesConfig `yaml:"services"`
	VKApi       VKApiConfig    `yaml:"vk_api"`
	GoogleOAuth GoogleOAuthConfig
	Gemini      GeminiConfig
	Proxies     ProxiesConfig

	Logger     LoggerConfig `yaml:"logger"`
	CtxUserKey string       `yaml:"user_key"`
	IsProd     bool         `yaml:"-"`
}

func ReadConfig(cfgPath string) *Config {
	cfg := &Config{}

	file, err := os.Open(cfgPath)
	if err != nil {
		log.Println("Something went wrong while opening config file ", err)

		return nil
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		log.Println("Something went wrong while reading config from yaml file ", err)

		return nil
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		log.Println("Something went wrong while reading config from env file ", err)

		return nil
	}

	log.Println("Successfully opened config")

	return cfg
}
