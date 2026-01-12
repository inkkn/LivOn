package config

import "time"

type Config struct {
	Service     *ServiceConfig
	Redis       *RedisConfig
	Postgres    *PostgresConfig
	Twilio      *TwilioConfig
	Worker      *WorkerConfig
	Logger      *LoggerConfig
	SecretToken string
}

type ServiceConfig struct {
	Name string
	Env  string
	Add  string
}

type RedisConfig struct {
	URL          string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	PingTimeout  time.Duration
}

type PostgresConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

type TwilioConfig struct {
	SID       string
	Token     string
	VerifySID string
}

type WorkerConfig struct {
	MessageGroup string
}

type LoggerConfig struct {
	Level  string
	Format string
}
