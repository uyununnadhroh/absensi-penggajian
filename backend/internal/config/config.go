package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Port        string   `yaml:"port"`
	Timezone    string   `yaml:"timezone"`
	DSN         string   `yaml:"dsn"`
	JWTSecret   string   `yaml:"jwt_secret"`
	CORSOrigins []string `yaml:"cors_origins"`
}

// Load membaca config.yaml (path dari env CONFIG_FILE, default ./config.yaml).
// Env DB_DSN, JWT_SECRET, PORT, APP_TZ, CORS_ORIGINS (dipisah koma) menimpa isi file.
func Load() (Config, error) {
	c := Config{Port: ":8080", Timezone: "Asia/Jakarta"}
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "config.yaml"
	}
	if b, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(b, &c); err != nil {
			return c, err
		}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		c.DSN = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		c.JWTSecret = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if !strings.HasPrefix(v, ":") {
			v = ":" + v
		}
		c.Port = v
	}
	if v := os.Getenv("APP_TZ"); v != "" {
		c.Timezone = v
	}
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		c.CORSOrigins = strings.Split(v, ",")
	}
	return c, nil
}
