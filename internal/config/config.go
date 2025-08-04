package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Freshdesk FreshdeskConfig
	Ollama    OllamaConfig
	RabbitMQ  RabbitMQConfig
	Database  DatabaseConfig
	Scheduler SchedulerConfig
	Telegram  TelegramConfig
	Email     EmailConfig
}

type FreshdeskConfig struct {
	Domain  string
	APIKey  string
	BaseURL string
}

type OllamaConfig struct {
	BaseURL      string
	ModelCS      string
	ModelSupport string
}

type RabbitMQConfig struct {
	URL       string
	QueueName string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type SchedulerConfig struct {
	Interval time.Duration
}

type TelegramConfig struct {
	BotToken string
	ChatID   string
}

type EmailConfig struct {
	SystemEmails []string
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		// Игнорируем ошибку если .env файл не найден
	}

	interval, _ := strconv.Atoi(getEnv("SCHEDULER_INTERVAL_MINUTES", "5"))

	config := &Config{
		Freshdesk: FreshdeskConfig{
			Domain:  getEnv("FRESHDESK_DOMAIN", ""),
			APIKey:  getEnv("FRESHDESK_API_KEY", ""),
			BaseURL: "https://" + getEnv("FRESHDESK_DOMAIN", "") + ".freshdesk.com/api/v2",
		},
		Ollama: OllamaConfig{
			BaseURL:      getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			ModelCS:      getEnv("OLLAMA_MODEL_CS", "CS_Bro"),
			ModelSupport: getEnv("OLLAMA_MODEL_SUPPORT", "Support_bro"),
		},
		RabbitMQ: RabbitMQConfig{
			URL:       getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
			QueueName: getEnv("RABBITMQ_QUEUE_NAME", "ticket_queue"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "ticket_system"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Scheduler: SchedulerConfig{
			Interval: time.Duration(interval) * time.Minute,
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
			ChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
		},
		Email: EmailConfig{
			SystemEmails: parseEmailList(getEnv("EMAIL_LIST", "noreply@,system@,admin@")),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func parseEmailList(emailList string) []string {
	if emailList == "" {
		return []string{}
	}
	return strings.Split(emailList, ",")
}
