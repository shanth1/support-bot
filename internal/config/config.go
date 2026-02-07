package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken     string
	AdminGroupID int64
	AppPort      string
	APIKey       string
	DBPath       string
}

func MustLoad() *Config {
	_ = godotenv.Load()

	adminID, _ := strconv.ParseInt(os.Getenv("ADMIN_GROUP_ID"), 10, 64)

	cfg := &Config{
		BotToken:     os.Getenv("BOT_TOKEN"),
		AdminGroupID: adminID,
		AppPort:      getEnv("PORT", "8088"),
		APIKey:       os.Getenv("API_KEY"),
		DBPath:       getEnv("DB_PATH", "./data/support.db"),
	}

	if cfg.BotToken == "" || cfg.AdminGroupID == 0 {
		log.Fatal("BOT_TOKEN and ADMIN_GROUP_ID are required")
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
