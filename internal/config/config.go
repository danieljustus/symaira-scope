// Package config loads symscope settings via corekit/configkit:
// ~/.config/symscope/config.toml with SYMSCOPE_* env overrides.
package config

import "github.com/danieljustus/symaira-corekit/configkit"

type Config struct {
	Ports PortsConfig `json:"ports" toml:"ports"`
}

type PortsConfig struct {
	SuggestFrom int `json:"suggest_from" toml:"suggest_from"`
	SuggestTo   int `json:"suggest_to" toml:"suggest_to"`
}

func Defaults() *Config {
	return &Config{
		Ports: PortsConfig{SuggestFrom: 3000, SuggestTo: 9999},
	}
}

var loader = configkit.NewLoader[Config](configkit.Options{
	AppName:   "symscope",
	EnvPrefix: "SYMSCOPE",
}, Defaults)

// Load returns the resolved config (defaults < config.toml < env).
func Load() (*Config, error) { return loader.Load() }
