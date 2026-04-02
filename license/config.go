package license

import "time"

type Config struct {
	APIURL       string
	ClientID     string
	ClientSecret string
	Interval     time.Duration
	Timeout      time.Duration
	StoragePath  string
	PublicKey    string
	GracePeriod  time.Duration
	MaxRetries   int
}

func (c Config) normalize() Config {
	out := c
	if out.Interval <= 0 {
		out.Interval = 30 * time.Second
	}
	if out.Timeout <= 0 {
		out.Timeout = 10 * time.Second
	}
	if out.GracePeriod <= 0 {
		out.GracePeriod = 10 * time.Minute
	}
	if out.MaxRetries <= 0 {
		out.MaxRetries = 3
	}
	return out
}
