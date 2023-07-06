package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func admin(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if len(apiKey) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ssion := state.DefaultSessionManager.Load(apiKey)
		if ssion == nil || !tools.Contains(config.Conf.Admins, ssion.ID) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func security(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if len(apiKey) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ssion := state.DefaultSessionManager.Load(apiKey)
		if ssion == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func common(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		w.Header().Add("X-Session-Valid", fmt.Sprintf("%t", len(apiKey) > 0 &&
			state.DefaultSessionManager.Load(apiKey) != nil))
		ssion := state.DefaultSessionManager.Load(apiKey)
		ctx := r.Context()
		if ssion != nil {
			ctx = context.WithValue(ctx, tools.KeySession, ssion)
			ctx = context.WithValue(ctx, tools.KeySessionUID, ssion.ID)
		} else {
			ctx = context.WithValue(ctx, tools.KeySessionUID, "")
		}
		h.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
