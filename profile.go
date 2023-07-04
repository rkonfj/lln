package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

var uniqueNameRegex = regexp.MustCompile(`^[\p{L}\d_]+$`)

type User struct {
	state.User
	Following  bool  `json:"following"`
	Followers  int64 `json:"followers"`
	Followings int64 `json:"followings"`
	Tweets     int64 `json:"tweets"`
}

func profile(w http.ResponseWriter, r *http.Request) {
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
	sessionUID := r.Context().Value(tools.KeySessionUID).(string)
	if r.Context().Value(tools.KeySessionUID) == nil || sessionUID != u.ID {
		// privacy
		u.Email = ""
		u.Locale = ""
	}

	json.NewEncoder(w).Encode(User{
		User:       *u,
		Followers:  u.Followers(),
		Followings: u.Followings(),
		Tweets:     u.Tweets(),
		Following:  u.FollowingBy(sessionUID)})
}

func followUser(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
	uniqueName, err := url.PathUnescape(chi.URLParam(r, tools.UniqueName))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = state.FollowUser(ssion.ToUser(), uniqueName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func modifyProfile(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
	u := state.UserByID(ssion.ID)
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	var mu state.ModifiableUser
	err := json.NewDecoder(r.Body).Decode(&mu)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if len(mu.Bio) > 0 {
		if utf8.RuneCountInString(mu.Bio) > 256 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "bio: maximum 256 unicode characters")
			return
		}
	}

	if len(mu.UniqueName) > 0 {
		if utf8.RuneCountInString(mu.UniqueName) > 12 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "unique name: maximum 12 unicode characters")
			return
		}
		if !uniqueNameRegex.MatchString(mu.UniqueName) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "unique name: not match `%s`", uniqueNameRegex)
			return
		}
		if tools.Contains(config.Model.Keywords, mu.UniqueName) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "unique name: %s is a keyword", mu.UniqueName)
			return
		}
	}

	if len(mu.Name) > 0 && utf8.RuneCountInString(mu.Name) > 20 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "name: maximum 20 unicode characters")
		return
	}

	if len(mu.Picture) > 0 && len(mu.Picture) > 256 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "picture: maximum 256 characters")
		return
	}

	if len(mu.Locale) > 0 && len(mu.Locale) > 6 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "locale: maximum 6 characters")
		return
	}

	err = u.Modify(mu)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	state.UserChanged <- u
	state.DefaultSessionManager.Expire(ssion.ID)
}
