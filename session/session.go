package session

import "github.com/rkonfj/lln/state"

var defaultSessionManager = NewSessionManager()

func Create(email, name, picture string) (*Session, error) {
	u := state.UserByEmail(email)
	if u == nil {
		u = state.NewUser(email, name, picture)
	}

	s := &Session{
		Name:       name,
		UniqueName: u.UniqueName,
		Picture:    picture,
	}
	defaultSessionManager.Create(s)
	return s, nil
}
