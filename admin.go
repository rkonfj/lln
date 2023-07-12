package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func userVerified(w http.ResponseWriter, r *http.Request) {
	u := state.UserByID(chi.URLParam(r, tools.UID))
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	code, err := tools.URLQueryInt64(r, "code")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	err = u.SetVerified(code)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	state.UserChanged <- u
	state.DefaultSessionManager.Expire(u.ID)
}

func recommendStatus(w http.ResponseWriter, r *http.Request) {
	if err := state.RecommendStatus(chi.URLParam(r, tools.StatusID)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
	}
}

func notRecommendStatus(w http.ResponseWriter, r *http.Request) {
	if err := state.NotRecommendStatus(chi.URLParam(r, tools.StatusID)); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
	}
}

func disableUser(w http.ResponseWriter, r *http.Request) {
	u := state.UserByID(chi.URLParam(r, tools.UID))
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	err := u.Disable()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}

func enableUser(w http.ResponseWriter, r *http.Request) {
	u := state.UserByID(chi.URLParam(r, tools.UID))
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	err := u.Enable()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}
