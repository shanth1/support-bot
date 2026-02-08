package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/shanth1/gotools/conf"
	"github.com/shanth1/gotools/consts"
	"github.com/shanth1/gotools/env"
	"github.com/shanth1/gotools/flags"
	"github.com/shanth1/gotools/log"
)

type Config struct {
	Env consts.Env `mapstructure:"env" env:"APP_ENV" validate:"required,oneof=local dev stage prod"`

	Logger log.Config `mapstructure:"logger" validate:"required"`

	Bot struct {
		Token         string        `mapstructure:"-" env:"BOT_TOKEN" validate:"required"`
		AdminGroupID  int64         `mapstructure:"admin_group_id" env:"BOT_ADMIN_GROUP_ID"`
		TopicID       int           `mapstructure:"topic_id" env:"BOT_TOPIC_ID"`
		PollerTimeout time.Duration `mapstructure:"poller_timeout" validate:"min=1s"`
	} `mapstructure:"bot" validate:"required"`

	Server struct {
		Enabled bool   `mapstructure:"enabled" env:"SERVER_ENABLED" validate:"required"`
		Addr    string `mapstructure:"addr" env:"SERVER_ADDR" validate:"required,hostname_port"`
		APIKey  string `mapstructure:"-" env:"SERVER_API_KEY"`
	} `mapstructure:"server" validate:"required"`

	Storage struct {
		Path          string `mapstructure:"path" env:"STORAGE_PATH" validate:"required"`
		RetentionDays int    `mapstructure:"retention_days" validate:"required,min=1"`
	} `mapstructure:"storage" validate:"required"`

	Messages Messages `mapstructure:"messages" validate:"required"`
}

type Messages struct {
	Start                   string `mapstructure:"start" validate:"required"`
	UserSentOk              string `mapstructure:"user_sent_ok" validate:"required"`
	AdminReplySent          string `mapstructure:"admin_reply_sent" validate:"required"`
	AdminNotificationHeader string `mapstructure:"admin_notification_header" validate:"required"`
	ErrorUserBlocked        string `mapstructure:"error_user_blocked" validate:"required"`
	ErrorCopyFailed         string `mapstructure:"error_copy_failed" validate:"required"`
	ErrorUserNotFound       string `mapstructure:"error_user_not_found" validate:"required"`
	ErrorGeneric            string `mapstructure:"error_generic" validate:"required"`
}

type bootstrapConfig struct {
	ConfigPath string `flag:"config" usage:"Path to the mapstructure config file"`
	EnvPath    string `flag:"env" usage:"Path to the env file"`
}

func Load() (*Config, error) {
	bootCfg := &bootstrapConfig{}
	if err := flags.RegisterFromStruct(bootCfg); err != nil {
		return nil, fmt.Errorf("register flags: %w", err)
	}
	flag.Parse()

	if _, err := os.Stat(bootCfg.ConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at path: %s", bootCfg.ConfigPath)
	}

	cfg := &Config{}
	if err := conf.Load(bootCfg.ConfigPath, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	if err := env.LoadIntoStruct(bootCfg.EnvPath, cfg); err != nil {
		return nil, fmt.Errorf("load env: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}
