package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/storage"
	"github.com/rs/xid"
)

func signRequest(w http.ResponseWriter, r *http.Request) {
	user := currentSessionUser(r)
	object := r.URL.Query().Get("object")
	if len(object) == 0 {
		object = base58.Encode(xid.New().Bytes())
	}

	if state.TodayMediaCountByUser(user) >= config.Conf.Model.Media.CountPerDayLimit {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(w, "upload up to %d media files per day", config.Conf.Model.Media.CountPerDayLimit)
		return
	}

	timePrefix := time.Now().Format("20060102")

	ns := fmt.Sprintf("%s/%s", timePrefix, user.ID)
	url, err := storage.S3SignRequest(ns, object)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	objectPath := fmt.Sprintf("/%s/%s", ns, object)
	if err = state.SaveMedia(user, objectPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	json.NewEncoder(w).Encode(R{V: map[string]string{
		"url":  url,
		"path": objectPath,
	}})
}
