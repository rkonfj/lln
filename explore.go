package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func explore(c *gin.Context) {
	sizeStr := c.Query("size")
	size := int64(20)
	var err error
	if len(sizeStr) != 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
	}
	var user *state.ActUser
	if s, ok := c.Get(util.KeySession); ok {
		user = s.(*session.Session).ToUser()
	}
	ss := state.Recommendations(user, c.Query("after"), size)
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
				Labels:     prev.Labels,
			}
		}
		ret = append(ret, status)
	}
	c.JSON(http.StatusOK, ret)
}
