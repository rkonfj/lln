package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

type StatusOptions struct {
	Content   []*state.StatusFragment `json:"content" binding:"required"`
	RefStatus string                  `json:"prev"`
}

type Status struct {
	ID         string                  `json:"id"`
	Content    []*state.StatusFragment `json:"content"`
	RefStatus  *Status                 `json:"prev"`
	User       *state.ActUser          `json:"user"`
	CreateTime time.Time               `json:"createTime"`
	Labels     []string                `json:"labels"`
	Comments   int64                   `json:"comments"`
	LikeCount  int64                   `json:"likeCount"`
	Views      int64                   `json:"views"`
}

var labelsRegex = regexp.MustCompile(`#([\p{L}\d_]+)`)
var imageRegex = regexp.MustCompile(`\[img\](https://[^\s\[\]]+)\[/img\]`)
var breaklineRegex = regexp.MustCompile(`\n\n+`)
var atRegex = regexp.MustCompile(`@([\p{L}\d_]+)`)

func status(w http.ResponseWriter, r *http.Request) {
	status := chainStatus(chi.URLParam(r, util.StatusID))
	if status == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(status)
}

func statusComments(w http.ResponseWriter, r *http.Request) {
	size := int64(20)
	sizeStr := r.URL.Query().Get("size")
	var err error
	if len(sizeStr) > 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	comments := state.StatusComments(chi.URLParam(r, util.StatusID), r.URL.Query().Get("after"), size)
	json.NewEncoder(w).Encode(comments)
}

func chainStatus(statusID string) *Status {
	status := state.GetStatus(statusID)
	if status == nil {
		return nil
	}
	s := castStatus(status)
	if len(status.RefStatus) > 0 {
		s.RefStatus = chainStatus(status.RefStatus)
	}
	return s
}

func userStatus(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	size := int64(20)
	var err error
	if len(sizeStr) > 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	ss := state.UserStatus(chi.URLParam(r, util.UniqueName), r.URL.Query().Get("after"), size)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s)
		prev := state.GetStatus(s.RefStatus)
		if prev != nil {
			status.RefStatus = castStatus(prev)
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(ret)
}

func likeStatus(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	err := state.LikeStatus(ssion.ToUser(), chi.URLParam(r, util.StatusID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func newStatus(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	req := &StatusOptions{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if req.Content == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("content is required"))
		return
	}

	for _, f := range req.Content {
		count := utf8.RuneCountInString(f.Value)
		if count > 380 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("maximum 380 unicode characters per paragraph, %d", count)))
			return
		}
	}

	opts := &state.StatusOptions{
		Content:   req.Content,
		RefStatus: req.RefStatus,
		User:      ssion.ToUser(),
		Labels:    []string{},
	}
	var sf []*state.StatusFragment
	for _, f := range opts.Content {
		if f.Type != "text" {
			continue
		}
		// process labels
		matches := labelsRegex.FindAllStringSubmatch(f.Value, -1)
		if len(matches) > 0 {
			for _, m := range matches {
				opts.Labels = append(opts.Labels, m[1])
			}
		}

		// process @
		atMatches := atRegex.FindAllStringSubmatch(f.Value, -1)
		if len(atMatches) > 0 {
			for _, m := range atMatches {
				opts.At = append(opts.At, m[1])
			}
		}

		// process breaklines
		f.Value = breaklineRegex.ReplaceAllString(f.Value, "\n\n")

		// process media images
		imgMatches := imageRegex.FindAllStringSubmatch(f.Value, -1)
		if len(imgMatches) > 0 {
			f.Type = "img"
			f.Value = imgMatches[0][1]
			for _, m := range imgMatches[1:] {
				sf = append(sf, &state.StatusFragment{
					Type:  "img",
					Value: m[1],
				})
			}
		}
	}

	opts.Content = append(opts.Content, sf...)

	s, err := state.NewStatus(opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	json.NewEncoder(w).Encode(s)
}

func castStatus(s *state.Status) *Status {
	return &Status{
		ID:         s.ID,
		Content:    s.Content,
		User:       s.User,
		CreateTime: s.CreateTime,
		Labels:     s.Labels,
		Comments:   s.Comments,
		Views:      s.Views,
		LikeCount:  s.LikeCount,
	}
}
