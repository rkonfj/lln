package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:     "lln",
		Short:   "A twitterlike api server",
		Args:    cobra.NoArgs,
		PreRunE: initAction,
		RunE:    startAction,
	}
	cmd.Flags().StringP("config", "c", "config.yml", "config file (default is config.yml)")
	cmd.Flags().String("log-level", logrus.InfoLevel.String(), "logging level")
	cmd.Execute()
}

func initAction(cmd *cobra.Command, args []string) error {
	logLevel, err := cmd.Flags().GetString("log-level")
	if err != nil {
		return err
	}
	ll, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	logrus.SetLevel(ll)
	logrus.SetOutput(os.Stdout)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	err = loadConfig(configPath)
	if err != nil {
		return err
	}

	err = state.InitState(state.EtcdOptions{
		Endpoints:     config.State.Etcd.Endpoints,
		CertFile:      config.State.Etcd.CertFile,
		KeyFile:       config.State.Etcd.KeyFile,
		TrustedCAFile: config.State.Etcd.TrustedCAFile,
	})
	if err != nil {
		return err
	}

	err = session.InitSession()
	return err
}

func startAction(cmd *cobra.Command, args []string) error {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Route("/i", func(r chi.Router) {
		r.Use(common, security)
		r.Post(fmt.Sprintf("/like/status/{%s}", util.StatusID), likeStatus)
		r.Post(fmt.Sprintf("/follow/user/{%s}", util.UniqueName), followUser)
		r.Post(fmt.Sprintf("/bookmark/status/{%s}", util.StatusID), bookmarkStatus)
		r.Post("/status", newStatus)
		r.Put("/profile", modifyProfile)
		r.Get("/bookmarks", listBookmarks)
		r.Get("/messages", listMessages)
		r.Get("/messages/tips", getNewTipMessages)
		r.Delete("/messages", deleteMessages)
		r.Delete("/messages/tips", deleteTipMessages)
		r.Delete("/authorize", deleteAuthorize)
		r.Delete(fmt.Sprintf("/status/{%s}", util.StatusID), deleteStatus)
	})

	r.Route("/o", func(r chi.Router) {
		r.Use(common)
		r.Post(fmt.Sprintf("/authorize/{%s}", util.Provider), authorize)
		r.Get(fmt.Sprintf("/authorize/{%s}", util.Provider), authorize)
		r.Get(fmt.Sprintf("/oidc/{%s}", util.Provider), oidcRedirect)
		r.Get(fmt.Sprintf("/user/{%s}", util.UniqueName), profile)
		r.Get(fmt.Sprintf("/user/{%s}/status", util.UniqueName), userStatus)
		r.Get(fmt.Sprintf("/status/{%s}", util.StatusID), status)
		r.Get(fmt.Sprintf("/status/{%s}/comments", util.StatusID), statusComments)
		r.Get("/search", search)
		r.Get("/explore", explore)
		r.Get("/labels", labels)
	})
	return http.ListenAndServe(config.Listen, r)
}

func security(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if len(apiKey) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ssion := session.DefaultSessionManager.Load(apiKey)
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
			session.DefaultSessionManager.Load(apiKey) != nil))
		ssion := session.DefaultSessionManager.Load(apiKey)
		ctx := r.Context()
		if ssion != nil {
			ctx = context.WithValue(ctx, util.KeySession, ssion)
			ctx = context.WithValue(ctx, util.KeySessionUID, ssion.ID)
		} else {
			ctx = context.WithValue(ctx, util.KeySessionUID, "")
		}
		h.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
