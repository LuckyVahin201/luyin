package config

import (
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
	RedirectURI string
	Providers   map[string]ProviderConfig
}

func New() Config {
	_ = godotenv.Load()

	return Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedirectURI: "http://localhost:3000/api/auth/callback",
		Providers: map[string]ProviderConfig{
			"trakt": {
				AuthURL:  "https://trakt.tv/oauth/authorize",
				TokenURL: "https://api.trakt.tv/oauth/token",
				ClientID: os.Getenv("TRAKT_CLIENT_ID"),
				UsePKCE:  true,
			},
			"anilist": {
				AuthURL:      "https://anilist.co/api/v2/oauth/authorize",
				TokenURL:     "https://anilist.co/api/v2/oauth/token",
				ClientID:     os.Getenv("ANILI_CLIENT_ID"),
				ClientSecret: os.Getenv("ANILI_CLIENT_SE"),
				UsePKCE:      false,
			},
		},
	}
}
