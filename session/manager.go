package session

import (
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/state"
	"github.com/rs/xid"
)

type Session struct {
	ApiKey     string `json:"apiKey"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	UniqueName string `json:"uniqueName"`
	Picture    string `json:"picture"`
}

func (s *Session) ToUser() *state.ActUser {
	return &state.ActUser{
		ID:         s.ID,
		Name:       s.Name,
		UniqueName: s.UniqueName,
		Picture:    s.Picture,
	}
}

type SessionManager interface {
	Create(*Session) error
	Load(string) *Session
	Expire(string) error
}

type MemorySessionManger struct {
	lock       sync.RWMutex
	session    map[string]*Session
	revSession map[string][]string
}

func NewSessionManager() *MemorySessionManger {
	return &MemorySessionManger{
		session:    make(map[string]*Session),
		revSession: make(map[string][]string),
	}
}

func (sm *MemorySessionManger) Create(s *Session) error {
	b := make([]byte, 16)
	rand.Reader.Read(b)
	s.ApiKey = fmt.Sprintf("sk-%s", base58.Encode(append(b, xid.New().Bytes()...)))
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.session[s.ApiKey] = s
	sm.revSession[s.ID] = append(sm.revSession[s.ID], s.ApiKey)
	return nil
}

func (sm *MemorySessionManger) Load(key string) *Session {
	return sm.session[key]
}

func (sm *MemorySessionManger) Expire(userID string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if apiKeys, ok := sm.revSession[userID]; ok {
		for _, key := range apiKeys {
			delete(sm.session, key)
		}
		delete(sm.revSession, userID)
	}
	return nil
}
