package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Значения по умолчанию для необязательных настроек.
const (
	defaultAppPort        = 8080             // порт HTTP-сервера
	defaultLogLevel       = "info"           // уровень логов
	defaultReadTimeout    = 10 * time.Second // таймаут чтения запроса
	defaultWriteTimeout   = 10 * time.Second // таймаут записи ответа
	defaultDBMaxLifetime  = 5 * time.Minute  // сколько живёт соединение с БД
	defaultDBMaxIdleTime  = 3 * time.Minute  // сколько простаивает соединение с БД
	defaultDBMaxOpenConns = 25               // максимум открытых соединений
	defaultDBMaxIdleConns = 10               // максимум простаивающих соединений
	defaultVenuePort      = 8081             // порт симулятора заведения
	defaultVenueInterval  = 30 * time.Second // как часто симулятор опрашивает заказы
	defaultVenueTimeout   = 10 * time.Second // таймаут запросов симулятора
)

// Config - все настройки приложения.
type Config struct {
	App      AppConfig      // настройки HTTP-сервера
	Database DatabaseConfig // настройки базы данных
}

// AppConfig - настройки HTTP-сервера.
type AppConfig struct {
	Port         int           // порт
	LogLevel     string        // уровень логов: debug, info, warn, error
	ReadTimeout  time.Duration // таймаут чтения запроса
	WriteTimeout time.Duration // таймаут записи ответа
}

// DatabaseConfig - настройки базы данных.
type DatabaseConfig struct {
	URL             string        // строка подключения, обязательна
	MaxOpenConns    int           // максимум открытых соединений
	MaxIdleConns    int           // максимум простаивающих соединений
	ConnMaxLifetime time.Duration // сколько живёт соединение
	ConnMaxIdleTime time.Duration // сколько простаивает соединение
}

// VenueConfig - настройки симулятора заведения.
type VenueConfig struct {
	Port           int           // свой порт
	APIKey         string        // ключ заведения для основного API
	MainAPIURL     string        // адрес основного сервиса
	PollInterval   time.Duration // как часто опрашивать заказы
	RequestTimeout time.Duration // таймаут запросов к основному API
}

// LoadVenue - читает настройки симулятора из env.
func LoadVenue() (*VenueConfig, error) {
	cfg := &VenueConfig{}

	var err error
	if cfg.Port, err = getEnvInt("VENUE_SERVICE_PORT", defaultVenuePort); err != nil {
		return nil, err
	}
	if cfg.PollInterval, err = getEnvDuration("VENUE_POLL_INTERVAL", defaultVenueInterval); err != nil {
		return nil, err
	}
	if cfg.RequestTimeout, err = getEnvDuration("VENUE_REQUEST_TIMEOUT", defaultVenueTimeout); err != nil {
		return nil, err
	}

	cfg.APIKey = getEnvStr("VENUE_API_KEY", "")
	cfg.MainAPIURL = getEnvStr("MAIN_API_URL", "")

	return cfg, nil
}

// Load - читает настройки из env, обязательные без значения дают ошибку.
func Load() (*Config, error) {
	cfg := &Config{}

	var err error
	if cfg.App.Port, err = getEnvInt("APP_PORT", defaultAppPort); err != nil {
		return nil, err
	}
	cfg.App.LogLevel = getEnvStr("APP_LOG_LEVEL", defaultLogLevel)
	if cfg.App.ReadTimeout, err = getEnvDuration("APP_READ_TIMEOUT", defaultReadTimeout); err != nil {
		return nil, err
	}
	if cfg.App.WriteTimeout, err = getEnvDuration("APP_WRITE_TIMEOUT", defaultWriteTimeout); err != nil {
		return nil, err
	}

	if cfg.Database.MaxOpenConns, err = getEnvInt("DB_MAX_OPEN_CONNS", defaultDBMaxOpenConns); err != nil {
		return nil, err
	}
	if cfg.Database.MaxIdleConns, err = getEnvInt("DB_MAX_IDLE_CONNS", defaultDBMaxIdleConns); err != nil {
		return nil, err
	}
	if cfg.Database.ConnMaxLifetime, err = getEnvDuration("DB_CONN_MAX_LIFETIME", defaultDBMaxLifetime); err != nil {
		return nil, err
	}
	if cfg.Database.ConnMaxIdleTime, err = getEnvDuration("DB_CONN_MAX_IDLE_TIME", defaultDBMaxIdleTime); err != nil {
		return nil, err
	}

	dbURL, err := getEnvRequired("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	cfg.Database.URL = dbURL

	return cfg, nil
}

// getEnvStr - берёт строку из env или возвращает fallback.
func getEnvStr(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// getEnvRequired - берёт обязательную строку из env, без неё возвращает ошибку.
func getEnvRequired(key string) (string, error) {
	if val := os.Getenv(key); val != "" {
		return val, nil
	}
	return "", fmt.Errorf("%s must be set", key)
}

// getEnvInt - берёт число из env, кривое значение даёт ошибку.
func getEnvInt(key string, fallback int) (int, error) {
	if val := os.Getenv(key); val != "" {
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("invalid %s %q: %w", key, val, err)
		}
		return i, nil
	}
	return fallback, nil
}

// getEnvDuration - берёт длительность из env, кривое значение даёт ошибку.
func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	if val := os.Getenv(key); val != "" {
		d, err := time.ParseDuration(val)
		if err != nil {
			return 0, fmt.Errorf("invalid %s %q: %w", key, val, err)
		}
		return d, nil
	}
	return fallback, nil
}
