package main

import (
	"encoding/json"
	"net/http"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func listMessages(w http.ResponseWriter, r *http.Request) {
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	msgs, more := state.ListMessages(currentSessionUser(r), opts)
	json.NewEncoder(w).Encode(L{V: msgs, More: more})
}

func deleteMessages(w http.ResponseWriter, r *http.Request) {
	abstractDeleteMessages(w, r, state.DeleteMessages)
}

func getNewTipMessages(w http.ResponseWriter, r *http.Request) {
	tipMsgs := state.ListTipMessages(currentSessionUser(r), 100)
	json.NewEncoder(w).Encode(L{V: tipMsgs})
}

func deleteTipMessages(w http.ResponseWriter, r *http.Request) {
	abstractDeleteMessages(w, r, state.DeleteTipMessages)
}

func abstractDeleteMessages(w http.ResponseWriter, r *http.Request, doDelete func(*state.ActUser, []string) error) {
	msgs := []string{}
	err := json.NewDecoder(r.Body).Decode(&msgs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = doDelete(currentSessionUser(r), msgs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}
