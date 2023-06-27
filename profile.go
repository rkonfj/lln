package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func profile(w http.ResponseWriter, r *http.Request) {
	uniqueName, err := url.PathUnescape(chi.URLParam(r, util.UniqueName))
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

	if r.Context().Value(util.KeySession) == nil {
		// privacy
		u.Email = ""
		u.Locale = ""
	}
	json.NewEncoder(w).Encode(u)
}

func likeUser(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	uniqueName, err := url.PathUnescape(chi.URLParam(r, util.UniqueName))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	err = state.LikeUser(ssion.ToUser(), uniqueName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

// changeName change name action or unique name action, only one action is atomic
func changeName(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	u := state.UserByID(ssion.ID)
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if len(name) > 0 {
		defer session.DefaultSessionManager.Expire(ssion.ID)
		err := u.ChangeName(name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	uniqueName := strings.TrimSpace(r.URL.Query().Get(util.UniqueName))
	if len(uniqueName) > 0 {
		defer session.DefaultSessionManager.Expire(ssion.ID)
		err := u.ChangeUniqueName(uniqueName)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		state.UserChanged <- u
		return
	}

	if len(name) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid name"))
	} else {
		state.UserChanged <- u
	}
}
