package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:     "lln",
		Short:   "A twitterlike api server",
		Version: fmt.Sprintf("%s, commit %s", tools.Version, tools.Commit),
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
	return err
}

func startAction(cmd *cobra.Command, args []string) error {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	r.Route("/i", func(r chi.Router) {
		r.Use(common, security)
		r.Post(fmt.Sprintf("/like/status/{%s}", tools.StatusID), likeStatus)
		r.Post(fmt.Sprintf("/follow/user/{%s}", tools.UniqueName), followUser)
		r.Post(fmt.Sprintf("/bookmark/status/{%s}", tools.StatusID), bookmarkStatus)
		r.Post("/status", newStatus)
		r.Put("/profile", modifyProfile)
		r.Get("/bookmarks", listBookmarks)
		r.Get("/messages", listMessages)
		r.Get("/messages/tips", getNewTipMessages)
		r.Delete("/messages", deleteMessages)
		r.Delete("/messages/tips", deleteTipMessages)
		r.Delete("/authorize", deleteAuthorize)
		r.Delete(fmt.Sprintf("/status/{%s}", tools.StatusID), deleteStatus)
	})

	r.Route("/o", func(r chi.Router) {
		r.Use(common)
		r.Post(fmt.Sprintf("/authorize/{%s}", tools.Provider), authorize)
		r.Get(fmt.Sprintf("/authorize/{%s}", tools.Provider), authorize)
		r.Get(fmt.Sprintf("/oidc/{%s}", tools.Provider), oidcRedirect)
		r.Get(fmt.Sprintf("/user/{%s}", tools.UniqueName), profile)
		r.Get(fmt.Sprintf("/user/{%s}/status", tools.UniqueName), userStatus)
		r.Get(fmt.Sprintf("/status/{%s}", tools.StatusID), status)
		r.Get(fmt.Sprintf("/status/{%s}/comments", tools.StatusID), statusComments)
		r.Get("/search", search)
		r.Get("/explore", explore)
		r.Get("/explore/news-probe", exploreNewsProbe)
		r.Get("/labels", labels)
	})
	logrus.Infof("listen %s for http now", config.Listen)
	return http.ListenAndServe(config.Listen, r)
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

type L struct {
	V    any    `json:"v"`
	Code uint16 `json:"code"`
	More bool   `json:"more"`
}

type R struct {
	V    any    `json:"v"`
	Code uint16 `json:"code"`
}
