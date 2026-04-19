package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
}

type ServerConfig struct {
	Host string `env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port string `env:"SERVER_PORT" env-default:"8080"`
}

type PostgresConfig struct {
	Host            string `env:"POSTGRES_HOST"            env-default:"db"`
	Port            string `env:"POSTGRES_PORT"            env-default:"5432"`
	User            string `env:"POSTGRES_USER"            env-default:"postgres"`
	Password        string `env:"POSTGRES_PASSWORD"        env-default:"postgres"`
	DB              string `env:"POSTGRES_DB"              env-default:"subscriptions"`
	SSLMode         string `env:"POSTGRES_SSLMODE"         env-default:"disable"`
	MaxConns        int32  `env:"POSTGRES_MAXCONNS"        env-default:"20"`
	MinConns        int32  `env:"POSTGRES_MINCONNS"        env-default:"2"`
	MaxConnLifetime int32  `env:"POSTGRES_MAXCONNLIFETIME" env-default:"30"`
	MaxConnIdleTime int32  `env:"POSTGRES_MAXCONNIDLE"     env-default:"5"`
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DB, c.SSLMode)
}

func (c PostgresConfig) MaxConnLifetimeDuration() time.Duration {
	return time.Duration(c.MaxConnLifetime) * time.Minute
}

func (c PostgresConfig) MaxConnIdleTimeDuration() time.Duration {
	return time.Duration(c.MaxConnIdleTime) * time.Minute
}

func Load() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return &cfg, nil
}
