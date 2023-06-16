package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func bookmarkStatus(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	err := state.BookmarkStatus(ssion.ToUser(), c.Param(util.StatusID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
}

func listBookmarks(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	size := int64(20)
	sizeStr := c.Query("size")
	var err error
	if len(sizeStr) > 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
	ss := state.ListBookmarks(ssion.ToUser(), c.Query("after"), size)
	c.JSON(http.StatusOK, ss)
}
