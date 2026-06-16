package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Kafka    KafkaConfig
	CH       ClickHouseConfig
	Short    ShortConfig
}

type AppConfig struct {
	Env      string `env:"APP_ENV"    env-default:"local"`
	Host     string `env:"APP_HOST"   env-default:"0.0.0.0"`
	Port     int    `env:"APP_PORT"   env-default:"8080"`
	GRPCPort int    `env:"GRPC_PORT"  env-default:"9090"`
	BaseURL  string `env:"BASE_URL"   env-default:"http://localhost:8080"`
}

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST"     env-default:"localhost"`
	Port     int    `env:"POSTGRES_PORT"     env-default:"5432"`
	User     string `env:"POSTGRES_USER"     env-default:"postgres"`
	Password string `env:"POSTGRES_PASSWORD" env-default:"postgres"`
	DB       string `env:"POSTGRES_DB"       env-default:"urlshortener"`
	SSLMode  string `env:"POSTGRES_SSL_MODE" env-default:"disable"`
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DB, c.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string        `env:"REDIS_ADDR"      env-default:"localhost:6379"`
	Password string        `env:"REDIS_PASSWORD"  env-default:""`
	DB       int           `env:"REDIS_DB"        env-default:"0"`
	CacheTTL time.Duration `env:"REDIS_CACHE_TTL" env-default:"24h"`
}

type KafkaConfig struct {
	Brokers     []string `env:"KAFKA_BROKERS"      env-default:"localhost:9092" env-separator:","`
	TopicClicks string   `env:"KAFKA_TOPIC_CLICKS" env-default:"url.clicks"`
	GroupID     string   `env:"KAFKA_GROUP_ID"     env-default:"url-shortener-workers"`
}

type ClickHouseConfig struct {
	Host     string `env:"CLICKHOUSE_HOST"     env-default:"localhost"`
	Port     int    `env:"CLICKHOUSE_PORT"     env-default:"9000"`
	DB       string `env:"CLICKHOUSE_DB"       env-default:"analytics"`
	User     string `env:"CLICKHOUSE_USER"     env-default:"default"`
	Password string `env:"CLICKHOUSE_PASSWORD" env-default:""`
}

type ShortConfig struct {
	CodeLength int `env:"SHORT_CODE_LENGTH" env-default:"7"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}
