package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
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
	cmd.Execute()
}

func initAction(cmd *cobra.Command, args []string) error {
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
	r.Use(security)
	r.POST(fmt.Sprintf("/authorize/:%s", Provider), authorize)
	r.POST(fmt.Sprintf("/like/status/:%s", StatusID), likeStatus)
	r.POST(fmt.Sprintf("/like/user/:%s", UniqueName), likeUser)
	r.POST("/status", newStatus)

	r.GET(fmt.Sprintf("/authorize/:%s", Provider), authorizeRedirect)
	r.GET(fmt.Sprintf("/:%s", UniqueName), profile)
	r.GET(fmt.Sprintf("/:%s/status", UniqueName), userStatus)
	r.GET(fmt.Sprintf("/status/:%s", StatusID), status)
	r.GET("/explore", explore)
	r.GET("/labels", labels)

	return r.Run(config.Listen)
}

func security(c *gin.Context) {
	if c.Request.RequestURI == "/status" {
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
		c.Set(KeySession, ssion)
	}
}
