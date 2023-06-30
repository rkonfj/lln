package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/decred/base58"
	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func authorize(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, util.Provider)
	provider := getProvider(providerName)
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

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("missing token"))
		return
	}

	// Parse and verify ID Token payload.
	idToken, err := provider.Provider.Verifier(&oidc.Config{ClientID: provider.Config.ClientID}).
		Verify(context.Background(), rawIDToken)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// Extract custom claims
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
		Picture  string `json:"picture"`
		Name     string `json:"name"`
		Locale   string `json:"locale"`
	}
	if err := idToken.Claims(&claims); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	if !claims.Verified {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unverified email"))
		return
	}
	sessionObj, err := session.Create(&state.UserOptions{
		Name:    claims.Name,
		Picture: claims.Picture,
		Email:   claims.Email,
		Locale:  claims.Locale,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	jump := string(base58.Decode(r.URL.Query().Get("state")))
	if r.Method == http.MethodPost {
		w.Header().Add("X-Jump", jump)
		json.NewEncoder(w).Encode(sessionObj)
	} else {
		http.Redirect(w, r, jump, http.StatusFound)
	}
}

func deleteAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Context().Value(util.KeySession) == nil {
		return
	}
	session.DefaultSessionManager.Delete(r.Header.Get("Authorization"))
}

func oidcRedirect(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, util.Provider)
	provider := getProvider(providerName)
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
