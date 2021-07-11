package hugo

import (
	"fmt"

	hugoconfig "github.com/gohugoio/hugo/config"
	"github.com/spf13/afero"
)

type Config struct {
	hugoconfig hugoconfig.Provider
}

func (c *Config) read() hugoconfig.Provider {
	if c.hugoconfig == nil {
		cfg, err := hugoconfig.FromFile(afero.NewOsFs(), "config.toml")
		if err != nil {
			fmt.Println("load config from file failed", err)
			return nil
		}
		c.hugoconfig = cfg
	}
	return c.hugoconfig
}

func (c *Config) GetBool(v string) bool {
	cfg := c.read()
	if cfg == nil || !cfg.IsSet(v) {
		fmt.Printf("config: no %v set, using default\n", v)
	}
	return cfg.GetBool(v)
}

func (c *Config) GetStringMapString(v string) map[string]string {
	cfg := c.read()
	if cfg == nil || !cfg.IsSet(v) {
		fmt.Printf("config: no %v set, using default\n", v)
	}
	return cfg.GetStringMapString(v)
}
