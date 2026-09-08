package config

import (
	"errors"
	"io/fs"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads a local .env file when present. Existing process environment
// variables take precedence, and a missing file is normal in containers/production.
func LoadDotEnv() error {
	err := godotenv.Load()
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
