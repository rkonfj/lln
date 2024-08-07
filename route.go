package main

import (
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/rkonfj/lln/config"
	"github.com/rkonfj/lln/tools"
)

func routeAdmin(r *chi.Mux) {
	r.Route("/v", func(r chi.Router) {
		r.Use(common, admin)
		r.Put(fmt.Sprintf("/user/{%s}/verified", tools.UID), userVerified)
		r.Put("/settings", putSettings)
		r.Post(fmt.Sprintf("/status/{%s}/recommend", tools.StatusID), recommendStatus)
		r.Delete(fmt.Sprintf("/status/{%s}/recommend", tools.StatusID), notRecommendStatus)
		r.Delete(fmt.Sprintf("/status/{%s}", tools.StatusID), deleteStatus)
		r.Post(fmt.Sprintf("/user/{%s}/disabled", tools.UID), disableUser)
		r.Delete(fmt.Sprintf("/user/{%s}/disabled", tools.UID), enableUser)
	})
}

func routeMustLogin(r *chi.Mux) {
	r.Route("/i", func(r chi.Router) {
		r.Use(common, security)
		r.Post(fmt.Sprintf("/like/status/{%s}", tools.StatusID), likeStatus)
		r.Post(fmt.Sprintf("/follow/user/{%s}", tools.UniqueName), followUser)
		r.Post(fmt.Sprintf("/bookmark/status/{%s}", tools.StatusID), bookmarkStatus)
		r.Post("/status", newStatus)
		r.Put("/profile", modifyProfile)
		r.Get("/bookmarks", listBookmarks)
		r.Get("/messages", listMessages)
		r.Get("/messages/tips", getNewTipMessages)
		r.Get("/restriction", config.GetRestriction)
		r.Get("/signed-upload-url", signRequest)
		r.Delete("/messages", deleteMessages)
		r.Delete("/messages/tips", deleteTipMessages)
		r.Delete("/authorize", deleteAuthorize)
		r.Delete(fmt.Sprintf("/status/{%s}", tools.StatusID), deleteStatus)
	})
}

func routeAnonymous(r *chi.Mux) {
	r.Route("/o", func(r chi.Router) {
		r.Use(common)
		r.Post(fmt.Sprintf("/authorize/{%s}", tools.Provider), authorize)
		r.Get(fmt.Sprintf("/authorize/{%s}", tools.Provider), authorize)
		r.Get(fmt.Sprintf("/oidc/{%s}", tools.Provider), oidcRedirect)
		r.Get(fmt.Sprintf("/user/{%s}", tools.UniqueName), profile)
		r.Get(fmt.Sprintf("/user/{%s}/status", tools.UniqueName), userStatus)
		r.Get(fmt.Sprintf("/status/{%s}", tools.StatusID), status)
		r.Get(fmt.Sprintf("/status/{%s}/comments", tools.StatusID), statusComments)
		r.Get(fmt.Sprintf("/explore/status/{%s}/comment", tools.StatusID), exploreStatusComment)
		r.Get("/search", search)
		r.Get("/explore", explore)
		r.Get("/explore/news-probe", exploreNewsProbe)
		r.Get("/labels", labels)
		r.Get("/settings", settings)
	})
}

func routeHTML(r *chi.Mux) {
	r.Get("/", exploreHTML)
	r.Get("/sitemap.xml", sitemap)
	r.Get("/explore", exploreHTML)
	r.Get("/friends", friendsHTML)
	r.Get(fmt.Sprintf("/{%s}", tools.UniqueName), profileHTML)
	r.Get(fmt.Sprintf("/{%s}/status/{%s}", tools.UniqueName, tools.StatusID), statusHTML)
}

type L struct {
	V    any    `json:"v"`
	Code uint16 `json:"code"`
	More bool   `json:"more"`
}

type R struct {
	V    any    `json:"v"`
	Code uint16 `json:"code"`
}
