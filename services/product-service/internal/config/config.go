package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

// Config utama dengan penambahan tag validasi
type Config struct {
	App      AppConfig      `koanf:"app" validate:"required"`
	Server   ServerConfig   `koanf:"server" validate:"required"`
	Database DatabaseConfig `koanf:"database" validate:"required"`
	Log      LogConfig      `koanf:"log" validate:"required"`
}

type AppConfig struct {
	Env  string `koanf:"env" validate:"required,oneof=development staging production"`
	Name string `koanf:"name" validate:"required"`
}

type ServerConfig struct {
	GRPCPort        int           `koanf:"grpc_port" validate:"required,min=1024,max=65535"`
	HTTPPort        int           `koanf:"http_port" validate:"required,min=1024,max=65535"`
	ReadTimeout     time.Duration `koanf:"read_timeout" validate:"required"`
	WriteTimeout    time.Duration `koanf:"write_timeout" validate:"required"`
	GracefulTimeout time.Duration `koanf:"graceful_timeout" validate:"required"`
}

type DatabaseConfig struct {
	Host        string        `koanf:"host" validate:"required"`
	Port        int           `koanf:"port" validate:"required,min=1,max=65535"`
	URL         string        `koanf:"url" validate:"required,uri"` // Validasi format URI
	MaxConns    int32         `koanf:"max_conns" validate:"required,min=1"`
	MinConns    int32         `koanf:"min_conns" validate:"min=0"`
	MaxIdleTime time.Duration `koanf:"max_idle_time" validate:"required"`
	MaxLifetime time.Duration `koanf:"max_lifetime" validate:"required"`
}

type LogConfig struct {
	Level string `koanf:"level" validate:"required,oneof=debug info warn error"`
}

func Load(configPath string) (*Config, error) {
	// 1. Load dari file YAML (hanya Warning jika gagal, karena mungkin murni pakai ENV)
	if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		fmt.Printf("Warning: Error loading yaml config: %v\n", err)
	}

	// 2. Load & Override dari Environment Variables
	err := k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("error loading env vars: %w", err)
	}

	// 3. Masukkan data ke Struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// 4. Proses Fail-Fast Validation (BARU)
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		// Jika ada konfigurasi krusial yang kosong/salah format, kembalikan error!
		return nil, fmt.Errorf("konfigurasi tidak valid: %w", err)
	}

	return &cfg, nil
}
