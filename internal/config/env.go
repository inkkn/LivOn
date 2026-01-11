package config

import (
	"time"
)

func Load() *Config {
	return &Config{
		Service: &ServiceConfig{
			Name: getEnv("SERVICE_NAME", "livon-backend"),
			Env:  getEnv("SERVICE_ENV", "development"),
			Add:  getEnv("SERVICE_ADDR", ":8080"),
		},
		Redis: &RedisConfig{
			URL:          getEnv("REDIS_URL", "redis://localhost:6379"),
			DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE", 2),
			PingTimeout:  getEnvDuration("REDIS_PING_TIMEOUT", 2*time.Second),
		},
		Postgres: &PostgresConfig{
			DSN:             getEnv("DATABASE_URL", "postgres://user:pass@localhost:5432/livon?sslmode=disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_LIFETIME", 15*time.Minute),
			ConnMaxIdleTime: getEnvDuration("DB_CONN_IDLE_TIME", 5*time.Minute),
			PingTimeout:     getEnvDuration("DB_PING_TIMEOUT", 5*time.Second),
		},
		Twilio: &TwilioConfig{
			SID:       getEnv("TWILIO_SID", ""),
			Token:     getEnv("TWILIO_TOKEN", ""),
			VerifySID: getEnv("TWILIO_VERIFY_SID", ""),
		},
		Worker: &WorkerConfig{
			MessageGroup: getEnv("WORKER_MESSAGE_GROUP", "conversation-workers"),
		},
		SecretToken: getEnv("JWT_SECRET", ""),
	}
}
