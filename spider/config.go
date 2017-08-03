package spider

import (
	"net/url"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type Config struct {
	Root         string `mapstructure:"root"`
	IgnoreRobots bool   `mapstructure:"ignore-robots"`
	RootURL      *url.URL
}

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
