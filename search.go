package main

import (
	"encoding/json"
	"net/http"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func search(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("type")
	value := r.URL.Query().Get("value")

	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var ss []*state.Status
	var more bool
	if t == "label" {
		ss, more = state.ListStatusByLabel(value, opts)
	} else {
		ss = state.ListStatusByKeyword(value, opts)
	}

	sessionUID := r.Context().Value(tools.KeySessionUID).(string)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s, sessionUID)
		if len(s.RefStatus) > 0 {
			prev := state.GetStatus(s.RefStatus)
			if prev != nil {
				status.RefStatus = castStatus(prev, sessionUID)
			}
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(L{V: ret, More: more})
}
