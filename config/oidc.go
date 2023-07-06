package config

import (
	"context"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var oidcProviders map[string]*OIDCProvider = make(map[string]*OIDCProvider)

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

func initOpenIDConnect() {
	var err error
	for _, o := range Conf.OIDC {
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

func GetOIDCProvider(provider string) *OIDCProvider {
	return oidcProviders[provider]
}
