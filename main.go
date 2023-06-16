package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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
	if ll != logrus.DebugLevel {
		gin.SetMode(gin.ReleaseMode)
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	err = loadConfig(configPath)
	if err != nil {
		return err
	}

	return state.InitState(state.EtcdOptions{
		Endpoints:     config.State.Etcd.Endpoints,
		CertFile:      config.State.Etcd.CertFile,
		KeyFile:       config.State.Etcd.KeyFile,
		TrustedCAFile: config.State.Etcd.TrustedCAFile,
	})
}

func startAction(cmd *cobra.Command, args []string) error {
	r := gin.Default()

	r.Group("/i").Use(security).Use(common).
		POST(fmt.Sprintf("/like/status/:%s", util.StatusID), likeStatus).
		POST(fmt.Sprintf("/like/user/:%s", util.UniqueName), likeUser).
		POST(fmt.Sprintf("/bookmark/status/:%s", util.StatusID), bookmarkStatus).
		POST("/status", newStatus).
		GET("/bookmarks", listBookmarks).
		GET("/messages", listMessages)

	r.Group("/o").Use(common).
		POST(fmt.Sprintf("/authorize/:%s", util.Provider), authorize).
		GET(fmt.Sprintf("/authorize/:%s", util.Provider), authorize).
		GET(fmt.Sprintf("/oidc/:%s", util.Provider), oidcRedirect).
		GET(fmt.Sprintf("/user/:%s", util.UniqueName), profile).
		GET(fmt.Sprintf("/user/:%s/status", util.UniqueName), userStatus).
		GET(fmt.Sprintf("/status/:%s", util.StatusID), status).
		GET(fmt.Sprintf("/status/:%s/comments", util.StatusID), statusComments).
		GET("/explore", explore).
		GET("/labels", labels)

	return r.Run(config.Listen)
}

func security(c *gin.Context) {
	apiKey := c.GetHeader("Authorization")
	if len(apiKey) == 0 {
		c.Status(http.StatusUnauthorized)
		c.Abort()
		return
	}
	ssion := session.DefaultSessionManager.Load(apiKey)
	if ssion == nil {
		c.Status(http.StatusUnauthorized)
		c.Abort()
		return
	}
	c.Set(util.KeySession, ssion)
}

func common(c *gin.Context) {
	apiKey := c.GetHeader("Authorization")
	if len(apiKey) == 0 {
		c.Header("X-Session-Valid", "false")
		return
	}
	ssion := session.DefaultSessionManager.Load(apiKey)
	c.Header("X-Session-Valid", fmt.Sprintf("%t", ssion != nil))
}
