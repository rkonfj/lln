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
	err := state.BookmarkStatus(currentSessionUser(r), chi.URLParam(r, tools.StatusID))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}

func listBookmarks(w http.ResponseWriter, r *http.Request) {
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	user := currentSessionUser(r)
	ss, more := state.ListBookmarks(user, opts)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s, user)
		if len(s.RefStatus) > 0 {
			prev := state.GetStatus(s.RefStatus)
			if prev != nil {
				status.RefStatus = castStatus(prev, user)
			}
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(L{V: ret, More: more})
}
