package config

import (
	"os"
	"path/filepath"
	"testing"
)

func chdirTemp(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func TestLoadDotEnvAllowsMissingFile(t *testing.T) {
	chdirTemp(t)
	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv() should allow absent .env: %v", err)
	}
}

func TestLoadDotEnvLoadsValuesWithoutOverridingProcessEnv(t *testing.T) {
	chdirTemp(t)
	const loadedKey = "FOOD_ORDERING_DOTENV_TEST_VALUE"
	old, existed := os.LookupEnv(loadedKey)
	if err := os.Unsetenv(loadedKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(loadedKey, old)
		} else {
			_ = os.Unsetenv(loadedKey)
		}
	})
	t.Setenv("APP_PORT", "8181")
	if err := os.WriteFile(filepath.Join(".", ".env"), []byte(loadedKey+"=loaded\nAPP_PORT=9999\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := LoadDotEnv(); err != nil {
		t.Fatalf("LoadDotEnv() error: %v", err)
	}
	if got := os.Getenv(loadedKey); got != "loaded" {
		t.Fatalf("loaded variable = %q, want loaded", got)
	}
	if got := os.Getenv("APP_PORT"); got != "8181" {
		t.Fatalf("existing environment variable was overwritten: got %q", got)
	}
}
