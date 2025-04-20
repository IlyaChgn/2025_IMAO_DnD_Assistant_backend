package config

import (
	"log"
	"os"

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

type ServiceConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type VKMethodConfig struct {
	URL    string `yaml:"url"`
	Method string `yaml:"method"`
}

type VKApiConfig struct {
	RedirectURI string         `env:"REDIRECT_URI"`
	ClientID    string         `env:"CLIENT_ID"`
	SecretKey   string         `env:"SECRET_KEY"`
	ServiceKey  string         `env:"SERVICE_KEY"`
	Exchange    VKMethodConfig `yaml:"exchange"`
	PublicInfo  VKMethodConfig `yaml:"public_info"`
}

type ServicesConfig struct {
	Description ServerConfig `yaml:"description"`
}

type Config struct {
	Server   ServerConfig `yaml:"server"`
	Mongo    MongoConfig
	Postgres PostgresConfig
	Services ServicesConfig `yaml:"services"`
	VKApi    VKApiConfig    `yaml:"vk_api"`
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
