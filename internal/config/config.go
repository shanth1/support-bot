package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"go.yaml.in/yaml/v2"
)

type Config struct {
	BotToken     string
	AdminGroupID int64
	APIKey       string
	Port         string
	DBPath       string

	Templates struct {
		AdminNotification string `yaml:"admin_notification"`
		UserForward       string `yaml:"user_forward"`
	} `yaml:"templates"`

	Messages struct {
		UserSentOk       string `yaml:"user_sent_ok"`
		AdminSentOk      string `yaml:"admin_sent_ok"`
		ErrorUserBlocked string `yaml:"error_user_blocked"`
		ErrorNotFound    string `yaml:"error_not_found"`
	} `yaml:"messages"`

	Projects []ProjectConfig `yaml:"projects"`
}

type ProjectConfig struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	TopicID  int    `yaml:"topic_id"`
	Greeting string `yaml:"greeting"`
}

func MustLoad() *Config {
	_ = godotenv.Load()

	langFlag := flag.String("lang", "", "Language to use (ru/en)")
	flag.Parse()

	lang := *langFlag
	if lang == "" {
		lang = os.Getenv("APP_LANG")
	}
	if lang == "" {
		lang = "ru"
	}

	configPath := filepath.Join("configs", fmt.Sprintf("%s.yaml", lang))
	file, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config for lang '%s': %v", lang, err))
	}

	var cfg Config
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		panic("Failed to parse YAML: " + err.Error())
	}

	cfg.BotToken = os.Getenv("BOT_TOKEN")
	cfg.APIKey = os.Getenv("API_KEY")
	cfg.Port = getEnv("PORT", "8088")
	cfg.DBPath = getEnv("DB_PATH", "./data/support.db")
	fmt.Sscanf(os.Getenv("ADMIN_GROUP_ID"), "%d", &cfg.AdminGroupID)

	if cfg.BotToken == "" {
		panic("BOT_TOKEN is not set")
	}

	return &cfg
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func (c *Config) GetProject(id string) ProjectConfig {
	for _, p := range c.Projects {
		if p.ID == id {
			return p
		}
	}
	return c.Projects[len(c.Projects)-1]
}
