package config

import (
	"context"
	"fmt"
	"github.com/sethvargo/go-envconfig"
	"net/url"
	"strconv"
)

type Config struct {
	Postgres PostgresConfig `env:",prefix=PSQL_" json:",omitempty"`
	Web      WebConfig      `env:",prefix=WEB_" json:",omitempty"`
	Service  ServiceConfig  `env:",prefix=SERVICE_" json:",omitempty"`
}

type PostgresConfig struct {
	Host        string `env:"HOST,default=localhost" json:",omitempty"`
	Port        int    `env:"PORT,default=5432" json:",omitempty"`
	Name        string `env:"NAME,default=postgres" json:",omitempty"`
	User        string `env:"USER,default=postgres" json:",omitempty"`
	Password    string `env:"PASSWORD,default=postgres" json:",omitempty"`
	SSLMode     string `env:"SSL_MODE,default=disable" json:",omitempty"`
	ConnTimeout int    `env:"CONN_TIMEOUT,default=5" json:",omitempty"`
}

type WebConfig struct {
	Host string `env:"HOST,default=localhost" json:",omitempty"`
	Port int    `env:"PORT,default=8080" json:",omitempty"`
}

type ServiceConfig struct {
	BaseCost      float64 `env:"BASE_COST,default=1000" json:",omitempty"`
	Interval      int     `env:"INTERVAL,default=3" json:",omitempty"`
	MaxRentPeriod int     `env:"MAX_RENT_PERIOD,default=30" json:",omitempty"`
}

func (p PostgresConfig) ConnectionURL() (string, error) {
	host := p.Host
	v := p.Port
	if v < 1 && v > 65536 {
		return "", fmt.Errorf("PSQL_PORT invalid")
	}
	host = host + ":" + strconv.Itoa(p.Port)

	u := &url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   p.Name,
	}

	if p.User == "" || p.Password == "" {
		return "", fmt.Errorf("PSQL_USER or PSQL_PASSWORD invalid")
	}
	u.User = url.UserPassword(p.User, p.Password)

	q := u.Query()
	connTimeout := p.ConnTimeout
	if connTimeout < 1 {
		return "", fmt.Errorf("PSQL_CONN_TIMEOUT invalid")
	}
	q.Add("connect_timeout", strconv.Itoa(p.ConnTimeout))

	if p.SSLMode != "disable" && p.SSLMode != "enable" {
		return "", fmt.Errorf("PSQL_SSL_MODE invalid")
	}
	q.Add("sslmode", p.SSLMode)

	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (w WebConfig) ConnectionURL() string {
	return fmt.Sprintf("%s:%d", w.Host, w.Port)
}

func Read(ctx context.Context) (Config, error) {
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return Config{}, fmt.Errorf("env processing: %w", err)
	}

	return cfg, nil

}
