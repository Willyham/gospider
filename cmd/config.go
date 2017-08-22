package cmd

import (
	"net/url"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

// Config holds all configuation needed to start a spider.
type Config struct {
	Root         string        `mapstructure:"root"`
	IgnoreRobots bool          `mapstructure:"ignore-robots"`
	Concurrency  int           `mapstructure:"concurrency"`
	Timeout      time.Duration `mapstructure:"timeout"`
	RootURL      *url.URL
}

// NewConfig creates a config from a deserialized map. Best used with
// viper.
func NewConfig(args map[string]interface{}) (*Config, error) {
	var conf Config
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.StringToTimeDurationHookFunc(),
		Result:     &conf,
	})
	if err != nil {
		return nil, err
	}

	err = dec.Decode(args)
	if err != nil {
		return nil, err
	}

	rootURL, err := url.Parse(conf.Root)
	if err != nil {
		return nil, errors.Wrap(err, "invalid root URL")
	}
	if rootURL.Scheme == "" || rootURL.Hostname() == "" {
		return nil, errors.New("invalid root URL")
	}
	conf.RootURL = rootURL

	return &conf, nil
}
