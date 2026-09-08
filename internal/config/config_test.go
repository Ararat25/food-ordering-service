package config

import (
	"testing"
)

func setBaseAppEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"APP_PORT": "8080", "APP_READ_TIMEOUT": "10s", "APP_WRITE_TIMEOUT": "10s",
		"DB_MAX_OPEN_CONNS": "25", "DB_MAX_IDLE_CONNS": "10",
		"DB_CONN_MAX_LIFETIME": "5m", "DB_CONN_MAX_IDLE_TIME": "3m",
		"DATABASE_URL": "postgres://user:pass@localhost:5432/db?sslmode=disable",
	} {
		t.Setenv(key, value)
	}
}

func setBaseVenueEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"VENUE_SERVICE_PORT": "8081", "VENUE_POLL_INTERVAL": "30s", "VENUE_REQUEST_TIMEOUT": "10s",
	} {
		t.Setenv(key, value)
	}
}

func TestLoadRejectsInvalidAppConfiguration(t *testing.T) {
	for _, tt := range []struct{ name, key, value string }{
		{"zero port", "APP_PORT", "0"},
		{"port above range", "APP_PORT", "65536"},
		{"zero read timeout", "APP_READ_TIMEOUT", "0s"},
		{"negative write timeout", "APP_WRITE_TIMEOUT", "-1s"},
		{"zero open connections", "DB_MAX_OPEN_CONNS", "0"},
		{"idle exceeds open", "DB_MAX_IDLE_CONNS", "26"},
		{"zero lifetime", "DB_CONN_MAX_LIFETIME", "0s"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setBaseAppEnv(t)
			t.Setenv(tt.key, tt.value)
			if _, err := Load(); err == nil {
				t.Fatalf("Load() succeeded with %s=%s", tt.key, tt.value)
			}
		})
	}
}

func TestLoadVenueRejectsUnsafeIntervalsAndPorts(t *testing.T) {
	for _, tt := range []struct{ name, key, value string }{
		{"zero port", "VENUE_SERVICE_PORT", "0"},
		{"port above range", "VENUE_SERVICE_PORT", "65536"},
		{"zero poll interval", "VENUE_POLL_INTERVAL", "0s"},
		{"negative poll interval", "VENUE_POLL_INTERVAL", "-1s"},
		{"zero request timeout", "VENUE_REQUEST_TIMEOUT", "0s"},
		{"negative request timeout", "VENUE_REQUEST_TIMEOUT", "-1s"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setBaseVenueEnv(t)
			t.Setenv(tt.key, tt.value)
			if _, err := LoadVenue(); err == nil {
				t.Fatalf("LoadVenue() succeeded with %s=%s", tt.key, tt.value)
			}
		})
	}
}
