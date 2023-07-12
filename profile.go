package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

type User struct {
	state.User
	Following  bool  `json:"following"`
	Followers  int64 `json:"followers"`
	Followings int64 `json:"followings"`
	Tweets     int64 `json:"tweets"`
	Disabled   bool  `json:"disabled"`
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
	user := currentSessionUser(r)
	if user == nil || user.ID != u.ID {
		// privacy
		u.Email = ""
		u.Locale = ""
	}

	json.NewEncoder(w).Encode(User{
		User:       *u,
		Followers:  u.Followers(),
		Followings: u.Followings(),
		Tweets:     u.Tweets(),
		Disabled:   u.Disabled(),
		Following:  u.FollowingBy(user)})
}

func followUser(w http.ResponseWriter, r *http.Request) {
	uniqueName, err := url.PathUnescape(chi.URLParam(r, tools.UniqueName))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = state.FollowUser(currentSessionUser(r), uniqueName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func modifyProfile(w http.ResponseWriter, r *http.Request) {
	u := state.UserByID(currentSessionUser(r).ID)
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	mu, err := checkArgsModifyProfile(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	err = u.Modify(*mu)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	state.UserChanged <- u
	state.DefaultSessionManager.Expire(currentSessionUser(r).ID)
}

func checkArgsModifyProfile(r *http.Request) (*state.ModifiableUser, error) {
	mu := &state.ModifiableUser{}
	err := json.NewDecoder(r.Body).Decode(mu)
	if err != nil {
		return nil, err
	}
	if len(mu.Bio) > 0 {
		if utf8.RuneCountInString(mu.Bio) > 256 {
			return nil, fmt.Errorf("bio: maximum 256 unicode characters")
		}
	}

	if len(mu.UniqueName) > 0 {
		if utf8.RuneCountInString(mu.UniqueName) > 12 {
			return nil, fmt.Errorf("unique name: maximum 12 unicode characters")
		}
		if !state.UniqueNameRegex.MatchString(mu.UniqueName) {
			return nil, fmt.Errorf("unique name: not match `%s`", state.UniqueNameRegex)
		}
		if tools.Contains(config.Conf.Model.Keywords, mu.UniqueName) {
			return nil, fmt.Errorf("unique name: %s is a keyword", mu.UniqueName)
		}
	}

	if len(mu.Name) > 0 && utf8.RuneCountInString(mu.Name) > 20 {
		return nil, fmt.Errorf("name: maximum 20 unicode characters")
	}

	if len(mu.Picture) > 0 && len(mu.Picture) > 256 {
		return nil, fmt.Errorf("picture: maximum 256 characters")
	}

	if len(mu.Locale) > 0 && len(mu.Locale) > 6 {
		return nil, fmt.Errorf("locale: maximum 6 characters")
	}
	return mu, nil
}
