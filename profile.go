package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func profile(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	}
	u := state.UserByUniqueName(c.Param(util.UniqueName))
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
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	err := state.LikeUser(ssion.ToUser(), c.Param(util.UniqueName))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
}

func changeName(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	u := state.UserByID(ssion.ID)
	if u == nil {
		c.JSON(http.StatusNotFound, "not found")
		return
	}
	name := c.Query("name")
	if len(name) == 0 {
		c.JSON(http.StatusBadRequest, "invalid name")
		return
	}
	err := u.ChangeName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	session.DefaultSessionManager.Expire(ssion.ID)
}

func changeUniqueName(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	u := state.UserByID(ssion.ID)
	if u == nil {
		c.JSON(http.StatusNotFound, "not found")
		return
	}
	name := c.Query(util.UniqueName)
	if len(name) == 0 {
		c.JSON(http.StatusBadRequest, "invalid name")
		return
	}
	err := u.ChangeUniqueName(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	session.DefaultSessionManager.Expire(ssion.ID)
}
