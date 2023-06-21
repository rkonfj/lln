package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func profile(w http.ResponseWriter, r *http.Request) {
	u := state.UserByUniqueName(r.URL.Query().Get(util.UniqueName))
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
	err := state.LikeUser(ssion.ToUser(), chi.URLParam(r, util.UniqueName))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func changeName(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	u := state.UserByID(ssion.ID)
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	name := r.URL.Query().Get("name")
	if len(name) > 0 {
		defer session.DefaultSessionManager.Expire(ssion.ID)
		err := u.ChangeName(name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	}

	uniqueName := r.URL.Query().Get("uniqueName")
	if len(uniqueName) > 0 {
		defer session.DefaultSessionManager.Expire(ssion.ID)
		err := u.ChangeUniqueName(name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		return
	}

	if len(name) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid name"))
	}

}
