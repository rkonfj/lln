package main

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:     "lln",
		Short:   "A twitterlike http api server",
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
	r.POST("/authorize/:provider", authorize)
	r.GET("/authorize/:provider", authorizeRedirect)
	r.GET("/:name", profile)
	r.GET("/:name/status", userStatus)
	r.GET("/:name/status/:id", status)
	r.GET("/explore", explore)
	r.GET("/labels", labels)
	r.POST("/like/status/:id", likeStatus)
	r.POST("/like/user/:name", likeUser)

	return r.Run(config.Listen)
}
