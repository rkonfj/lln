package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func listMessages(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	after := r.URL.Query().Get("after")
	sizeStr := r.URL.Query().Get("size")
	size := int64(20)
	if len(sizeStr) > 0 {
		var err error
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	msgs := state.ListMessages(ssion.ToUser(), after, size)
	json.NewEncoder(w).Encode(msgs)
}

func deleteMessages(w http.ResponseWriter, r *http.Request) {
	abstractDeleteMessages(w, r, state.DeleteMessages)
}

func getNewTipMessages(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	tipMsgs := state.ListTipMessages(ssion.ToUser(), 100)
	json.NewEncoder(w).Encode(tipMsgs)
}

func deleteTipMessages(w http.ResponseWriter, r *http.Request) {
	abstractDeleteMessages(w, r, state.DeleteTipMessages)
}

func abstractDeleteMessages(w http.ResponseWriter, r *http.Request, doDelete func(*state.ActUser, []string) error) {
	var ssion = r.Context().Value(util.KeySession).(*session.Session)
	msgs := []string{}
	err := json.NewDecoder(r.Body).Decode(&msgs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = doDelete(ssion.ToUser(), msgs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}
