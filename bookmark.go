package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func bookmarkStatus(w http.ResponseWriter, r *http.Request) {
	ssion := r.Context().Value(tools.KeySession).(*state.Session)
	err := state.BookmarkStatus(ssion.ToUser(), chi.URLParam(r, tools.StatusID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func listBookmarks(w http.ResponseWriter, r *http.Request) {
	ssion := r.Context().Value(tools.KeySession).(*state.Session)
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	ss, more := state.ListBookmarks(ssion.ToUser(), opts)
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
