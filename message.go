package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func listMessages(c *gin.Context) {
	c.JSON(http.StatusOK, nil)
}
