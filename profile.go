package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
)

func profile(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(KeySession); ok {
		ssion = s.(*session.Session)
	}
	u := state.UserByUniqueName(c.Param(UniqueName))
	if u == nil {
		c.Status(http.StatusNotFound)
		return
	}
	if ssion == nil {
		// privacy
		u.Email = ""
		u.Locale = ""
	}
	c.JSON(http.StatusOK, u)
}

func likeUser(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	err := state.LikeUser(ssion.ToUser(), c.Param(UniqueName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
}
