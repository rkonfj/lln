package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/state"
)

func labels(c *gin.Context) {
	labels := state.GetLabels(c.Query("prefix"))
	c.JSON(http.StatusOK, labels)
}
