package oidcauth

type Config struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	Scopes       string // space-separated; defaults to "openid email profile"
}

func DefaultConfig() Config {
	return Config{Scopes: "openid email profile"}
}

func (c Config) IsConfigured() bool {
	return c.Issuer != "" && c.ClientID != "" && c.ClientSecret != ""
}
