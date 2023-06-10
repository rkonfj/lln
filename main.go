package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
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
	return loadConfig(configPath)
}

func startAction(cmd *cobra.Command, args []string) error {
	r := gin.Default()
	r.POST(fmt.Sprintf("/authorize/:%s", Provider), authorize)
	r.POST(fmt.Sprintf("/like/status/:%s", StatusID), likeStatus)
	r.POST(fmt.Sprintf("/like/user/:%s", UniqueName), likeUser)

	r.GET(fmt.Sprintf("/authorize/:%s", Provider), authorizeRedirect)
	r.GET(fmt.Sprintf("/:%s", UniqueName), profile)
	r.GET(fmt.Sprintf("/:%s/status", UniqueName), userStatus)
	r.GET(fmt.Sprintf("/:%s/status/:%s", UniqueName, StatusID), status)
	r.GET("/explore", explore)
	r.GET("/labels", labels)

	return r.Run(config.Listen)
}
