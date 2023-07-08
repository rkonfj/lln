package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/state"
)

type Settings struct {
	state.Settings
	Status        config.StatusConfig `json:"status"`
	OIDCProviders []string            `json:"oidcProviders"`
}

func settings(w http.ResponseWriter, r *http.Request) {
	s, err := state.GetSettings()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if fmt.Sprintf("%d", s.ModRev) == r.URL.Query().Get("modRev") {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	err = json.NewEncoder(w).Encode(R{V: Settings{
		Settings:      *s,
		OIDCProviders: config.OIDCProviders(),
		Status:        config.Conf.Model.Status,
	}})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}

func putSettings(w http.ResponseWriter, r *http.Request) {
	s := state.Settings{}
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	err = state.UpdateSettings(&s)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}
