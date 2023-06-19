package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/decred/base58"
	"github.com/gin-gonic/gin"
	"github.com/rkonfj/lln/session"
	"github.com/rkonfj/lln/state"
	"github.com/rkonfj/lln/util"
)

func authorize(c *gin.Context) {
	providerName := c.Param(util.Provider)
	provider := getProvider(providerName)
	if provider == nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("provider %s not supported", providerName))
		return
	}
	oauth2Token, err := provider.Config.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, "missing token")
		return
	}

	// Parse and verify ID Token payload.
	idToken, err := provider.Provider.Verifier(&oidc.Config{ClientID: provider.Config.ClientID}).
		Verify(context.Background(), rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}

	// Extract custom claims
	var claims struct {
		Email    string `json:"email"`
		Verified bool   `json:"email_verified"`
		Picture  string `json:"picture"`
		Name     string `json:"name"`
		Locale   string `json:"locale"`
	}
	if err := idToken.Claims(&claims); err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	if !claims.Verified {
		c.JSON(http.StatusInternalServerError, "unverified email")
		return
	}
	sessionObj, err := session.Create(&state.UserOptions{
		Name:    claims.Name,
		Picture: claims.Picture,
		Email:   claims.Email,
		Locale:  claims.Locale,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	jump := string(base58.Decode(c.Query("state")))
	if c.Request.Method == http.MethodPost {
		c.Header("X-Jump", jump)
		c.JSON(http.StatusOK, sessionObj)
	} else {
		c.Redirect(http.StatusFound, jump)
	}
}

func oidcRedirect(c *gin.Context) {
	providerName := c.Param(util.Provider)
	provider := getProvider(providerName)
	if provider == nil {
		c.JSON(http.StatusBadRequest, fmt.Sprintf("provider %s not supported", providerName))
		return
	}

	jump := c.Query("jump")
	if len(jump) == 0 {
		jump = "/"
	}

	c.Redirect(http.StatusFound, provider.Config.AuthCodeURL(base58.Encode([]byte(jump))))
}
