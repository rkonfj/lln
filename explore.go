package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/tools"
)

func explore(w http.ResponseWriter, r *http.Request) {
	opts, err := tools.URLPaginationOptions(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}

	var user *state.ActUser
	if r.Context().Value(tools.KeySession) != nil {
		user = r.Context().Value(tools.KeySession).(*state.Session).ToUser()
	}

	sessionUID := r.Context().Value(tools.KeySessionUID).(string)
	ss, more := state.Recommendations(user, opts)
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

func exploreNewsProbe(w http.ResponseWriter, r *http.Request) {
	after, err := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	var user *state.ActUser
	if r.Context().Value(tools.KeySession) != nil {
		user = r.Context().Value(tools.KeySession).(*state.Session).ToUser()
	}
	json.NewEncoder(w).Encode(R{V: state.RecommendCount(user, after)})
}
