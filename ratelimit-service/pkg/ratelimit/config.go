package ratelimit

import "time"

type Config struct {
	Domain      string                `yaml:"domain"`
	Separator   string                `yaml:"separator"`
	Descriptors []RateLimitDescriptor `yaml:"descriptors"`
}

type RateLimitDescriptor struct {
	Key         string                `yaml:"key"`
	Value       string                `yaml:"value,omitempty"`
	RateLimit   *RateLimitValue       `yaml:"rate_limit,omitempty"`
	Descriptors []RateLimitDescriptor `yaml:"descriptors,omitempty"`
}

type RateLimitValue struct {
	Unit            string `yaml:"unit"`
	RequestsPerUnit int    `yaml:"requests_per_unit"`
}

type LimitKey struct {
	FullKey    string
	Domain     string
	Components map[string]string
	Timestamp  int64
	LimitValue int
	Unit       string
	TTL        time.Duration
}

func GetDefaultConfig() *Config {
    return &Config{
        Domain:    "ratelimit",
        Separator: "|",
        Descriptors: []RateLimitDescriptor{
            {
                Key: "",
                RateLimit: &RateLimitValue{
                    Unit:            "minute",
                    RequestsPerUnit: 60,
                },
            },
            {
                Key:   "path",
                Value: "/test",
                RateLimit: &RateLimitValue{
                    Unit:            "second",
                    RequestsPerUnit: 10,
                },
                Descriptors: []RateLimitDescriptor{
                    {
                        Key: "user_id",
                        RateLimit: &RateLimitValue{
                            Unit:            "minute",
                            RequestsPerUnit: 2,
                        },
                    },
                },
            },
        },
    }
}