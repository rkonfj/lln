package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func explore(w http.ResponseWriter, r *http.Request) {
	sizeStr := r.URL.Query().Get("size")
	size := int64(20)
	var err error
	if len(sizeStr) != 0 {
		size, err = strconv.ParseInt(sizeStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	var user *state.ActUser
	if r.Context().Value(util.KeySession) != nil {
		user = r.Context().Value(util.KeySession).(*session.Session).ToUser()
	}
	ss := state.Recommendations(user, r.URL.Query().Get("after"), size)
	var ret []*Status
	for _, s := range ss {
		status := castStatus(s)
		prev := state.GetStatus(s.RefStatus)
		if prev != nil {
			status.RefStatus = castStatus(prev)
		}
		ret = append(ret, status)
	}
	json.NewEncoder(w).Encode(ret)
}
