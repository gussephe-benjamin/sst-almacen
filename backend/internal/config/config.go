package config

import "os"

type Config struct {
	DatabaseURL string
	JWTSecret   string
	UploadDir   string
	Port        string
}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func Load() Config {
	return Config{
		DatabaseURL: get("DATABASE_URL", "postgres://sst:sst@localhost:5433/sst?sslmode=disable"),
		JWTSecret:   get("JWT_SECRET", "dev-secret-cambiar"),
		UploadDir:   get("UPLOAD_DIR", "./data/uploads"),
		Port:        get("PORT", "8081"),
	}
}
