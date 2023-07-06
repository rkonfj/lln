package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/decred/base58"
	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
)

func authorize(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, tools.Provider)
	provider := config.GetOIDCProvider(providerName)
	if provider == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("provider %s not supported", providerName)))
		return
	}

	oauth2Token, err := provider.Config.Exchange(context.Background(), r.URL.Query().Get("code"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	u, err := provider.Provider.UserInfo(context.Background(),
		provider.Config.TokenSource(context.Background(), oauth2Token))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "get userInfo error: %s", err.Error())
		return
	}

	if !provider.TrustEmail && !u.EmailVerified {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unverified email"))
		return
	}

	profile := make(map[string]any)

	err = u.Claims(&profile)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	logrus.Debugf("oauth2 user info: %s", profile)

	opts := &state.UserOptions{}
	if v, ok := profile[provider.UserMeta.Name]; ok && v != nil {
		opts.Name = v.(string)
	}
	if v, ok := profile[provider.UserMeta.Picture]; ok && v != nil {
		opts.Picture = v.(string)
	}
	if v, ok := profile[provider.UserMeta.Email]; ok && v != nil {
		opts.Email = v.(string)
	}
	if v, ok := profile[provider.UserMeta.Locale]; ok && v != nil {
		opts.Locale = v.(string)
	}
	if v, ok := profile[provider.UserMeta.Bio]; ok && v != nil {
		opts.Bio = v.(string)
	}

	if len(opts.Email) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "email is required")
		return
	}

	sessionObj, err := state.CreateSession(opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "create session error: %s", err)
		return
	}
	jump := string(base58.Decode(r.URL.Query().Get("state")))
	if r.Method == http.MethodPost {
		w.Header().Add("X-Jump", jump)
		json.NewEncoder(w).Encode(R{V: sessionObj})
	} else {
		http.Redirect(w, r, jump, http.StatusFound)
	}
}

func deleteAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(tools.KeySession) == nil {
		return
	}
	state.DefaultSessionManager.Delete(r.Header.Get("Authorization"))
}

func oidcRedirect(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, tools.Provider)
	provider := config.GetOIDCProvider(providerName)
	if provider == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("provider %s not supported", providerName)))
		return
	}

	jump := r.URL.Query().Get("jump")
	if len(jump) == 0 {
		jump = "/"
	}
	http.Redirect(w, r, provider.Config.AuthCodeURL(base58.Encode([]byte(jump))), http.StatusFound)
}
