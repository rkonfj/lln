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
	Content   []state.StatusFragment `json:"content" binding:"required"`
	RefStatus string                 `json:"prev"`
}

type Status struct {
	ID         string                 `json:"id"`
	Content    []state.StatusFragment `json:"content"`
	RefStatus  *Status                `json:"prev"`
	User       *state.ActUser         `json:"user"`
	CreateTime time.Time              `json:"createTime"`
	Labels     []string               `json:"labels"`
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

func statusComments(c *gin.Context) {
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
	comments := state.StatusComments(c.Param(util.StatusID), c.Query("after"), size)
	c.JSON(http.StatusOK, comments)
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
	if len(sizeStr) > 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, err.Error())
			return
		}
	}
	ss := state.UserStatus(c.Param(util.UniqueName), c.Query("after"), size)
	var ret []*Status
	for _, s := range ss {
		status := &Status{
			ID:         s.ID,
			Content:    s.Content,
			User:       s.User,
			CreateTime: s.CreateTime,
			Labels:     s.Labels,
		}
		prev := state.GetStatus(s.RefStatus)
		if prev != nil {
			status.RefStatus = &Status{
				ID:         prev.ID,
				Content:    prev.Content,
				User:       prev.User,
				CreateTime: prev.CreateTime,
			}
		}
		ret = append(ret, status)
	}
	c.JSON(http.StatusOK, ret)
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
	for _, f := range opts.Content {
		if f.Type != "text" {
			continue
		}
		matches := labelsRegex.FindAllStringSubmatch(f.Value, -1)
		if len(matches) > 0 {
			for _, m := range matches {
				opts.Labels = append(opts.Labels, m[1])
			}
		}
	}

	s, err := state.NewStatus(opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, s)
}
