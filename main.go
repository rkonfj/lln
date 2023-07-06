package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rkonfj/lln/config"
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
		PreRunE: initDaemon,
		RunE:    startDeamon,
	}
	cmd.Flags().StringP("config", "c", "config.yml", "config file (default is config.yml)")
	cmd.Flags().String("log-level", logrus.InfoLevel.String(), "logging level")
	cmd.Execute()
}

func initDaemon(cmd *cobra.Command, args []string) error {
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

	// init config
	err = config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	// init state
	err = state.InitState(state.EtcdOptions{
		Endpoints:     config.Conf.State.Etcd.Endpoints,
		CertFile:      config.Conf.State.Etcd.CertFile,
		KeyFile:       config.Conf.State.Etcd.KeyFile,
		TrustedCAFile: config.Conf.State.Etcd.TrustedCAFile,
	})
	return err
}

func startDeamon(cmd *cobra.Command, args []string) error {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	routeAnonymous(r)
	routeMustLogin(r)
	routeAdmin(r)

	logrus.Infof("listen %s for http now", config.Conf.Listen)
	return http.ListenAndServe(config.Conf.Listen, r)
}
