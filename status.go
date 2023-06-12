package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
)

type StatusOptions struct {
	Content   string `json:"content" binding:"required"`
	RefStatus string `json:"prev"`
}

type Status struct {
	ID         string         `json:"id"`
	Content    string         `json:"content"`
	RefStatus  *Status        `json:"prev"`
	User       *state.ActUser `json:"user"`
	CreateTime time.Time      `json:"createTime"`
}

func status(c *gin.Context) {
	status := chainStatus(c.Param(StatusID))
	if status == nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, status)
}

func chainStatus(statusID string) *Status {
	status := state.GetStatus(statusID)
	if status == nil {
		return nil
	}
	s := &Status{
		ID:         status.ID,
		Content:    status.Content,
		User:       status.User,
		CreateTime: status.CreateTime,
	}
	if len(status.RefStatus) > 0 {
		s.RefStatus = chainStatus(status.RefStatus)
	}
	return s
}

func userStatus(c *gin.Context) {
	sizeStr := c.Query("size")
	size := int64(10)
	var err error
	if len(sizeStr) == 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, state.UserStatus(c.Param(UniqueName), c.Query("after"), size))
}

func likeStatus(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	err := state.LikeStatus(ssion.ToUser(), c.Param(StatusID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
}

func newStatus(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	opts := &StatusOptions{}
	err := c.BindJSON(opts)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	s, err := state.NewStatus(&state.StatusOptions{
		Content:   opts.Content,
		RefStatus: opts.RefStatus,
		User:      ssion.ToUser(),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, s)
}
