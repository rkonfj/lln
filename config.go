package main

import (
	"context"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string      `yaml:"listen"`
	OIDC   []OIDC      `yaml:"oidc"`
	State  StateConfig `yaml:"state"`
	Model  ModelConfig `yaml:"model"`
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

type StateConfig struct {
	Etcd *EtcdConfig `yaml:"etcd"`
}
type EtcdConfig struct {
	Endpoints     []string `yaml:"endpoints"`
	CertFile      string   `yaml:"certFile,omitempty"`
	KeyFile       string   `yaml:"keyFile,omitempty"`
	TrustedCAFile string   `yaml:"trustedCAFile,omitempty"`
}

type ModelConfig struct {
	Status StatusConfig `yaml:"status"`
}

type StatusConfig struct {
	ContentListLimit int `yaml:"contentListLimit"`
	ContentLimit     int `yaml:"contentLimit"`
}

var (
	config        *Config
	oidcProviders map[string]*OIDCProvider = make(map[string]*OIDCProvider)
)

func loadConfig(configPath string) error {
	logrus.Info("loading config from ", configPath)
	config = &Config{}
	configF, err := os.Open(configPath)
	if err != nil {
		return err
	}
	err = yaml.NewDecoder(configF).Decode(config)
	if err != nil {
		return err
	}
	if config.State.Etcd == nil {
		config.State.Etcd = &EtcdConfig{Endpoints: []string{"http://127.0.0.1:2379"}}
	}

	if config.Model.Status.ContentLimit == 0 {
		config.Model.Status.ContentLimit = 380
	}

	if config.Model.Status.ContentListLimit == 0 {
		config.Model.Status.ContentListLimit = 20
	}

	for _, o := range config.OIDC {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		provider, err := oidc.NewProvider(ctx, o.Issuer)
		if err != nil {
			logrus.Error("oidc component error: ", err)
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
