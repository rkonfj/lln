package main

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
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

var labelsRegex = regexp.MustCompile(`#([\p{L}\d_]+)`)

func status(c *gin.Context) {
	status := chainStatus(c.Param(util.StatusID))
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
	size := int64(20)
	var err error
	if len(sizeStr) == 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
	c.JSON(http.StatusOK, state.UserStatus(c.Param(util.UniqueName), c.Query("after"), size))
}

func likeStatus(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	err := state.LikeStatus(ssion.ToUser(), c.Param(util.StatusID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
	}
}

func newStatus(c *gin.Context) {
	var ssion *session.Session
	if s, ok := c.Get(util.KeySession); ok {
		ssion = s.(*session.Session)
	} else {
		c.Status(http.StatusUnauthorized)
		return
	}
	req := &StatusOptions{}
	err := c.BindJSON(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	opts := &state.StatusOptions{
		Content:   req.Content,
		RefStatus: req.RefStatus,
		User:      ssion.ToUser(),
		Labels:    []string{},
	}
	matches := labelsRegex.FindAllStringSubmatch(opts.Content, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			opts.Labels = append(opts.Labels, m[1])
		}
	}
	s, err := state.NewStatus(opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, s)
}
