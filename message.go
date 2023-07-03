package main

import (
	"encoding/json"
	"net/http"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func listMessages(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	msgs, more := state.ListMessages(ssion.ToUser(), opts)
	json.NewEncoder(w).Encode(L{V: msgs, More: more})
}

func deleteMessages(w http.ResponseWriter, r *http.Request) {
	abstractDeleteMessages(w, r, state.DeleteMessages)
}

func getNewTipMessages(w http.ResponseWriter, r *http.Request) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
	tipMsgs := state.ListTipMessages(ssion.ToUser(), 100)
	json.NewEncoder(w).Encode(L{V: tipMsgs})
}

func deleteTipMessages(w http.ResponseWriter, r *http.Request) {
	abstractDeleteMessages(w, r, state.DeleteTipMessages)
}

func abstractDeleteMessages(w http.ResponseWriter, r *http.Request, doDelete func(*state.ActUser, []string) error) {
	var ssion = r.Context().Value(tools.KeySession).(*state.Session)
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
