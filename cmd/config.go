package cmd

import (
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// Config holds all configuation needed to start a spider.
type Config struct {
	Root         string `mapstructure:"root"`
	IgnoreRobots bool   `mapstructure:"ignore-robots"`
	RootURL      *url.URL
}

// NewConfig creates a config from a deserialized map. Best used with
// viper.
func NewConfig(args map[string]interface{}) (*Config, error) {
	var conf Config
	err := mapstructure.Decode(args, &conf)
	if err != nil {
		return nil, err
	}

	rootURL, err := url.Parse(conf.Root)
	if err != nil || rootURL.Scheme == "" || rootURL.Hostname() == "" {
		return nil, errors.Wrap(err, "invalid root URL")
	}
	conf.RootURL = rootURL

	return &conf, nil
}
