package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"unicode/utf8"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen string      `yaml:"listen"`
	OIDC   []*OIDC     `yaml:"oidc"`
	State  StateConfig `yaml:"state"`
	Model  ModelConfig `yaml:"model"`
	Admins []string    `yaml:"admins"`
}

type OIDC struct {
	Provider     string   `yaml:"provider"`
	Issuer       string   `yaml:"issuer"`
	AuthURL      string   `yaml:"authURL"`
	TokenURL     string   `yaml:"tokenURL"`
	UserInfoURL  string   `yaml:"userInfoURL"`
	ClientID     string   `yaml:"clientID"`
	ClientSecret string   `yaml:"clientSecret"`
	Redirect     string   `yaml:"redirect"`
	Scopes       []string `yaml:"scopes"`
	TrustEmail   bool     `yaml:"trustEmail"`
	UserMeta     UserMeta `yaml:"userMeta"`
}

type UserMeta struct {
	Email   string `yaml:"email"`
	Name    string `yaml:"name"`
	Picture string `yaml:"picture"`
	Bio     string `yaml:"bio"`
	Locale  string `yaml:"locale"`
}

type OIDCProvider struct {
	Provider   *oidc.Provider
	Config     *oauth2.Config
	TrustEmail bool
	UserMeta   *UserMeta
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
	ContentListLimit int `yaml:"contentListLimit" json:"contentListLimit"`
	ContentLimit     int `yaml:"contentLimit" json:"contentLimit"`
	OverviewLimit    int `yaml:"overviewLimit" json:"overviewLimit"`
}

func (c *StatusConfig) RestrictContent(content string) error {
	count := utf8.RuneCountInString(content)
	if count > c.ContentLimit {
		return fmt.Errorf("maximum %d unicode characters per paragraph, %d",
			c.ContentLimit, count)
	}
	return nil
}

func (c *StatusConfig) RestrictContentList(contentListSize int) error {
	if contentListSize > config.Model.Status.ContentListLimit {
		return fmt.Errorf("maximum %d content blocks, %d",
			c.ContentListLimit, contentListSize)
	}
	return nil
}

func (c *StatusConfig) RestrictOverview(content string) error {
	count := utf8.RuneCountInString(content)
	if count > c.OverviewLimit {
		return fmt.Errorf("maximum %d unicode characters in status overview, %d",
			c.OverviewLimit, count)
	}
	return nil
}

var (
	config        *Config
	oidcProviders map[string]*OIDCProvider = make(map[string]*OIDCProvider)
)

func loadConfig(configPath string) error {
	logrus.Info("loading config from ", configPath)
	config = &Config{}
	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(os.ExpandEnv(string(b))), config)
	if err != nil {
		return err
	}

	if config.State.Etcd == nil {
		config.State.Etcd = &EtcdConfig{Endpoints: []string{"http://127.0.0.1:2379"}}
	}

	if config.Model.Status.OverviewLimit == 0 {
		config.Model.Status.OverviewLimit = 256
	}

	if config.Model.Status.ContentLimit == 0 {
		config.Model.Status.ContentLimit = 4096
	}

	if config.Model.Status.ContentListLimit == 0 {
		config.Model.Status.ContentListLimit = 20
	}

	initOpenIDConnect()

	return err
}

func initOpenIDConnect() {
	var err error
	for _, o := range config.OIDC {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var provider *oidc.Provider
		if len(o.Issuer) > 0 {
			provider, err = oidc.NewProvider(ctx, o.Issuer)
			if err != nil {
				logrus.Error("oidc component error: ", err)
				continue
			}
		} else {
			provider = (&oidc.ProviderConfig{
				AuthURL:     o.AuthURL,
				TokenURL:    o.TokenURL,
				UserInfoURL: o.UserInfoURL,
			}).NewProvider(ctx)
		}

		if len(o.UserMeta.Picture) == 0 {
			o.UserMeta.Picture = "picture"
		}

		if len(o.UserMeta.Name) == 0 {
			o.UserMeta.Name = "name"
		}

		if len(o.UserMeta.Email) == 0 {
			o.UserMeta.Email = "email"
		}

		if len(o.UserMeta.Bio) == 0 {
			o.UserMeta.Bio = "bio"
		}

		if len(o.UserMeta.Locale) == 0 {
			o.UserMeta.Locale = "locale"
		}

		oidcProviders[o.Provider] = &OIDCProvider{
			Provider: provider,
			Config: &oauth2.Config{
				ClientID:     o.ClientID,
				ClientSecret: o.ClientSecret,
				RedirectURL:  o.Redirect,
				Endpoint:     provider.Endpoint(),
				Scopes:       o.Scopes,
			},
			TrustEmail: o.TrustEmail,
			UserMeta:   &o.UserMeta,
		}
	}
}

func getProvider(provider string) *OIDCProvider {
	return oidcProviders[provider]
}

type Restriction struct {
	Status StatusConfig `json:"status"`
}

func getRestriction(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(Restriction{
		Status: config.Model.Status,
	})
}
