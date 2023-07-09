package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
)

type StatusOptions struct {
	Content   []*state.StatusFragment `json:"content" binding:"required"`
	RefStatus string                  `json:"prev"`
}

type Status struct {
	ID         string                  `json:"id"`
	Content    []*state.StatusFragment `json:"content"`
	RefStatus  *Status                 `json:"prev"`
	Next       *Status                 `json:"next"`
	User       *state.ActUser          `json:"user"`
	CreateRev  int64                   `json:"createRev"`
	CreateTime time.Time               `json:"createTime"`
	Comments   int64                   `json:"comments"`
	LikeCount  int64                   `json:"likeCount"`
	Views      int64                   `json:"views"`
	Bookmarks  int64                   `json:"bookmarks"`
	Liked      bool                    `json:"liked"`
	Bookmarked bool                    `json:"bookmarked"`
	Followed   bool                    `json:"followed"`
	Disabled   bool                    `json:"disabled"`
}

func (s *Status) Overview() string {
	for _, c := range s.Content {
		if c.Type == "text" {
			return strings.TrimSpace(strings.ReplaceAll(c.Value, "\n", ""))
		}
	}
	return ""
}

var labelsRegex = regexp.MustCompile(`#([\p{L}\d_]+)`)
var imageRegex = regexp.MustCompile(`\[img\](https://[^\s\[\]]+)\[/img\]`)
var breaklineRegex = regexp.MustCompile(`\n\n+`)
var atRegex = regexp.MustCompile(`@([\p{L}\d_]+)`)

// status thread model
func status(w http.ResponseWriter, r *http.Request) {
	statusID := chi.URLParam(r, tools.StatusID)
	status := chainStatus(statusID, currentSessionUser(r))
	if status == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	state.SVCM.Viewed(statusID)
	logrus.WithField("ip", r.RemoteAddr).
		WithField("status", statusID).
		WithField("ua", r.Header.Get("User-Agent")).Info()
	json.NewEncoder(w).Encode(status)
}

// statusComments thread comments model
func statusComments(w http.ResponseWriter, r *http.Request) {
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	comments, more := state.StatusComments(chi.URLParam(r, tools.StatusID), opts)
	user := currentSessionUser(r)
	var ss []*Status
	for _, c := range comments {
		s := castStatus(c, user)
		ss = append(ss, s)
		meta, err := state.NewCommentsRecommandMeta(s.ID)
		if err != nil {
			logrus.Error(err)
			continue
		}
		rc := meta.Recommand()
		if rc != nil {
			s.Next = castStatus(rc, user)
		}
	}
	json.NewEncoder(w).Encode(L{V: ss, More: more})
}

func chainStatus(statusID string, sessionUser *state.ActUser) *Status {
	status := state.GetStatus(statusID)
	if status == nil {
		return nil
	}
	s := castStatus(status, sessionUser)
	if len(status.RefStatus) > 0 {
		s.RefStatus = chainStatus(status.RefStatus, sessionUser)
	}
	return s
}

func userStatus(w http.ResponseWriter, r *http.Request) {
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	uniqueName, err := url.PathUnescape(chi.URLParam(r, tools.UniqueName))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	u := state.UserByUniqueName(uniqueName)
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ss, more := u.ListStatus(opts)
	user := currentSessionUser(r)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s, user)
		if len(s.RefStatus) > 0 {
			prev := state.GetStatus(s.RefStatus)
			if prev != nil {
				status.RefStatus = castStatus(prev, user)
			}
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(L{V: ret, More: more})
}

func likeStatus(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
	err := state.LikeStatus(ssion.ToUser(), chi.URLParam(r, tools.StatusID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func newStatus(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
	req := &StatusOptions{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if req.Content == nil || len(req.Content) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("content is required"))
		return
	}

	if err := config.Conf.Model.Status.RestrictContentList(len(req.Content)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	for _, f := range req.Content {
		if err := config.Conf.Model.Status.RestrictContent(f.Value); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	opts := &state.StatusOptions{
		Content:   req.Content,
		RefStatus: req.RefStatus,
		User:      ssion.ToUser(),
		Labels:    []string{},
	}
	var overviewRestricted bool
	var sf []*state.StatusFragment
	for _, f := range opts.Content {
		if f.Type != "text" {
			continue
		}
		if !overviewRestricted {
			if err := config.Conf.Model.Status.RestrictOverview(f.Value); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, err.Error())
				return
			}
			overviewRestricted = true
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

	imgCount := 0
	for _, c := range opts.Content {
		if c.Type == "img" {
			imgCount++
		}
	}

	if imgCount > 4 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("maximum 4 images, %d", imgCount)))
		return
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

func deleteStatus(w http.ResponseWriter, r *http.Request) {
	s := state.GetStatus(chi.URLParam(r, tools.StatusID))
	if s == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, f := range s.ContentsByType("text") {
		matches := labelsRegex.FindAllStringSubmatch(f.Value, -1)
		if len(matches) > 0 {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, state.ErrStatusQuotes)
			return
		}
	}

	err := s.Delete(r.Context().Value(tools.KeySessionUID).(string))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
}
