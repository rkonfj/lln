package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/templates"
	"github.com/rkonfj/lln/tools"
)

var (
	statusTemplate  *template.Template
	profileTemplate *template.Template
	exploreTemplate *template.Template
)

func init() {
	t, err := template.New("status").Funcs(template.FuncMap{
		"last": func(index int, data any) bool {
			slice := data.([]*Status)
			return index == len(slice)-1
		},
	}).Parse(templates.Status)
	if err != nil {
		panic(err)
	}
	statusTemplate = t

	t, err = template.New("profile").Parse(templates.Profile)
	if err != nil {
		panic(err)
	}
	profileTemplate = t

	t, err = template.New("explore").Parse(templates.Explore)
	if err != nil {
		panic(err)
	}
	exploreTemplate = t
}

func statusHTML(w http.ResponseWriter, r *http.Request) {
	uniqueName, _ := url.PathUnescape(chi.URLParam(r, tools.UniqueName))
	statusID := chi.URLParam(r, tools.StatusID)
	s := chainStatus(statusID, nil)
	if s == nil || s.User.UniqueName != uniqueName {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, templates.NotFound)
		return
	}

	cur := s
	var ss []*Status
	ss = append(ss, cur)

	for {
		cur = cur.RefStatus
		if cur == nil {
			break
		}
		cur.Content = []*state.StatusFragment{
			{Type: "text", Value: cur.Overview()},
		}
		ss = append(ss, cur)
	}

	tools.Reverse(ss)

	comments, _ := state.StatusComments(statusID, &tools.PaginationOptions{
		Size:   100,
		Ascend: true,
	})

	for _, cur := range comments {
		cur.Content = []*state.StatusFragment{
			{Type: "text", Value: cur.Overview()},
		}
	}

	if err := statusTemplate.Execute(w, map[string]any{
		"overview": s.Overview(),
		"list":     ss,
		"comments": comments,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}

func profileHTML(w http.ResponseWriter, r *http.Request) {
	uniqueName, _ := url.PathUnescape(chi.URLParam(r, tools.UniqueName))
	u := state.UserByUniqueName(uniqueName)
	if u == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, templates.NotFound)
		return
	}

	ss, _ := u.ListStatus(&tools.PaginationOptions{Size: 100})

	for _, cur := range ss {
		cur.Content = []*state.StatusFragment{
			{Type: "text", Value: cur.Overview()},
		}
	}

	if err := profileTemplate.Execute(w, map[string]any{
		"profile": u,
		"list":    ss,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}

func exploreHTML(w http.ResponseWriter, r *http.Request) {
	ss, _ := state.Recommendations(nil, &tools.PaginationOptions{
		Size: 100,
	})
	if ss == nil {
		return
	}

	for _, cur := range ss {
		cur.Content = []*state.StatusFragment{
			{Type: "text", Value: cur.Overview()},
		}
	}

	if err := exploreTemplate.Execute(w, ss); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
	}
}
