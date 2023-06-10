package main

import (
	"context"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string `yaml:"listen"`
	OIDC   []OIDC `yaml:"oidc"`
}

type OIDC struct {
	Provider     string   `yaml:"provider"`
	Issuer       string   `yaml:"issuer"`
	ClientID     string   `yaml:"clientID"`
	ClientSecret string   `yaml:"clientSecret"`
	Redirect     string   `yaml:"redirect"`
	Scopes       []string `yaml:"scopes"`
}

type OIDCProvider struct {
	Provider *oidc.Provider
	Config   *oauth2.Config
}

var (
	config        *Config
	oidcProviders map[string]*OIDCProvider = make(map[string]*OIDCProvider)
)

func loadConfig(configPath string) error {
	config = &Config{}
	configF, err := os.Open(configPath)
	if err != nil {
		return err
	}
	err = yaml.NewDecoder(configF).Decode(config)
	if err != nil {
		return err
	}
	for _, o := range config.OIDC {
		provider, err := oidc.NewProvider(context.Background(), o.Issuer)
		if err != nil {
			logrus.Error(err)
			continue
		}

		oidcProviders[o.Provider] = &OIDCProvider{
			Provider: provider,
			Config: &oauth2.Config{
				ClientID:     o.ClientID,
				ClientSecret: o.ClientSecret,
				RedirectURL:  o.Redirect,
				// Discovery returns the OAuth2 endpoints.
				Endpoint: provider.Endpoint(),
				// "openid" is a required scope for OpenID Connect flows.
				Scopes: append(o.Scopes, oidc.ScopeOpenID),
			},
		}
	}
	return err
}

func getProvider(provider string) *OIDCProvider {
	return oidcProviders[provider]
}
