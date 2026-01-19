package githubauth

type Config struct {
	ClientID     string
	ClientSecret string
}

func DefaultConfig() Config {
	return Config{}
}

func (c Config) IsConfigured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}
