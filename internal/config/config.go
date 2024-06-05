package config

import (
	"fmt"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
	"os"
)

const (
	Webhook           UpdaterKind = "webhook"
	WebhookCustomCert UpdaterKind = "webhook_with_custom_cert"
	LongPolling       UpdaterKind = "long_polling"
)

const (
	Stable EnvType = "stable"
	Dev    EnvType = "dev"
)

type (
	Config struct {
		Telegram Telegram `yaml:"telegram"`
		DB       Database `yaml:"db"`
		Server   Server   `yaml:"server"`
		EnvType  EnvType  `yaml:"env"`
	}

	UpdaterKind string
	EnvType     string

	Server struct {
		Address string `yaml:"address" valid:"required" envconfig:"SERVER_ADDRESS"`
		Port    int    `yaml:"port" valid:"required" envconfig:"SERVER_PORT"`
		Secret  string `yaml:"secret" valid:"required" envconfig:"SERVER_SECRET"`
	}

	Telegram struct {
		Token       string      `yaml:"token" valid:"required" envconfig:"TELEGRAM_TOKEN"`
		UpdaterKind UpdaterKind `yaml:"updaterKind" valid:"required" envconfig:"TELEGRAM_UPDATER_KIND"`

		WebhookUrl    string `yaml:"webhook_url" valid:"required" envconfig:"TELEGRAM_WEBHOOK_URL"`
		WebhookListen string `yaml:"webhook_listen" valid:"required" envconfig:"TELEGRAM_WEBHOOK_LISTEN"`
		WebHookTSLKey string `yaml:"webhook_tls_key" valid:"required" envconfig:"TELEGRAM_WEBHOOK_TLS_KEY"`
		WebHookTSLCrt string `yaml:"webhook_tls_cert" valid:"required" envconfig:"TELEGRAM_WEBHOOK_TLS_CERT"`
	}

	Database struct {
		Host         string `yaml:"host" envconfig:"DB_HOST"`
		Port         int    `yaml:"port" envconfig:"DB_PORT"`
		DatabaseName string `yaml:"database_name" envconfig:"DB_DATABASE_NAME"`
		Username     string `yaml:"username" envconfig:"DB_USERNAME"`
		Password     string `yaml:"password" envconfig:"DB_PASSWORD"`
	}
)

func Load(path string) (*Config, error) {
	config := &Config{}

	if err := fromYaml(path, config); err != nil {
		fmt.Printf("couldn'n load config from %s: %s\r\n", path, err.Error())
	}

	if err := fromEnv(config); err != nil {
		fmt.Printf("couldn'n load config from env: %s\r\n", err.Error())
	}

	return config, nil
}

func fromYaml(path string, config *Config) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

func fromEnv(config *Config) error {
	return envconfig.Process("", config)
}
