package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type ProviderConfig struct {
	AuthURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	UsePKCE      bool
}

type Config struct {
	DatabaseURL string
	JWTSecret   string
	RedirectURI string
	Providers   map[string]ProviderConfig
}

func New() Config {
	_ = godotenv.Load()

	level := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	return Config{
		DatabaseURL: require("DATABASE_URL"),
		JWTSecret:   require("JWT_SECRET"),
		RedirectURI: "http://localhost:3000/api/auth/callback",
		Providers: map[string]ProviderConfig{
			"trakt": {
				AuthURL:  "https://trakt.tv/oauth/authorize",
				TokenURL: "https://api.trakt.tv/oauth/token",
				ClientID: require("TRAKT_CLIENT_ID"),
				UsePKCE:  true,
			},
			"anilist": {
				AuthURL:      "https://anilist.co/api/v2/oauth/authorize",
				TokenURL:     "https://anilist.co/api/v2/oauth/token",
				ClientID:     require("ANILI_CLIENT_ID"),
				ClientSecret: require("ANILI_CLIENT_SE"),
				UsePKCE:      false,
			},
		},
	}
}

func require(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required environment variable: " + key)
	}

	return v
}
