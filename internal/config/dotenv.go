package config

import (
	"errors"
	"io/fs"

	"github.com/joho/godotenv"
)

// LoadDotEnv загружает локальный файл .env, если он присутствует.
func LoadDotEnv() error {
	err := godotenv.Load()
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
