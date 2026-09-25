// Package config читает настройки из переменных окружения. Секреты в коде и репозитории не хранятся.
package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL string
	APIAddr     string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		APIAddr:     os.Getenv("API_ADDR"),
	}
	if c.APIAddr == "" {
		c.APIAddr = ":8080"
	}
	if c.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL не задан")
	}
	return c, nil
}
