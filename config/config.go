package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	DB       DBConfig
	Redis    RedisConfig
	LogLevel string
}

type ServerConfig struct {
	HTTPPort string
	GRPCPort string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Addr     string
	Password string
}

func (d DBConfig) DSN() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.Name + "?sslmode=" + d.SSLMode
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()
	_ = viper.ReadInConfig()

	viper.SetDefault("SERVER_HTTP_PORT", "8080")
	viper.SetDefault("SERVER_GRPC_PORT", "9090")
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_USER", "gotrack")
	viper.SetDefault("DB_PASSWORD", "gotrack_secret")
	viper.SetDefault("DB_NAME", "gotrack")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("LOG_LEVEL", "info")

	cfg := &Config{
		Server: ServerConfig{
			HTTPPort: viper.GetString("SERVER_HTTP_PORT"),
			GRPCPort: viper.GetString("SERVER_GRPC_PORT"),
		},
		DB: DBConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetString("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_NAME"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("REDIS_ADDR"),
			Password: viper.GetString("REDIS_PASSWORD"),
		},
		LogLevel: viper.GetString("LOG_LEVEL"),
	}

	if cfg.Server.HTTPPort == "" {
		return nil, fmt.Errorf("SERVER_HTTP_PORT is required")
	}
	if cfg.DB.Host == "" {
		return nil, fmt.Errorf("DB_HOST is required")
	}
	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("REDIS_ADDR is required")
	}

	return cfg, nil
}
