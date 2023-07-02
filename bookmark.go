package main

import (
	"encoding/json"
	"net/http"
	"strconv"

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
	size := int64(20)
	sizeStr := r.URL.Query().Get("size")
	var err error
	if len(sizeStr) > 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	ss, more := state.ListBookmarks(ssion.ToUser(), r.URL.Query().Get("after"), size)
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
