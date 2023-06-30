package session

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/state"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
)

type Session struct {
	ApiKey     string `json:"apiKey"`
	ID         string `json:"id"`
	Name       string `json:"name"`
	UniqueName string `json:"uniqueName"`
	Picture    string `json:"picture"`
	Locale     string `json:"locale"`
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
	Delete(string) error
	Expire(string) error
}

type MemorySessionManger struct {
	lock       sync.RWMutex
	session    map[string]*Session
	revSession map[string][]string
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

func (sm *MemorySessionManger) Delete(apiKey string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	s := sm.session[apiKey]
	if s == nil {
		return nil
	}
	delete(sm.session, apiKey)

	apiKeys := []string{}

	for _, key := range sm.revSession[s.ID] {
		if key != apiKey {
			apiKeys = append(apiKeys, key)
		}
	}

	sm.revSession[s.ID] = apiKeys
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

type PersistentSessionManager struct {
	MemorySessionManger
}

func NewSessionManager() *PersistentSessionManager {
	sm := &PersistentSessionManager{
		MemorySessionManger: MemorySessionManger{
			session:    make(map[string]*Session),
			revSession: make(map[string][]string),
		},
	}
	err := state.IterateWithPrefix("/session", func(_ string, value []byte) {
		ssion := Session{}
		err := json.Unmarshal(value, &ssion)
		if err != nil {
			logrus.Error(err)
			return
		}
		sm.MemorySessionManger.session[ssion.ApiKey] = &ssion
		sm.MemorySessionManger.revSession[ssion.ID] = append(sm.MemorySessionManger.revSession[ssion.ID], ssion.ApiKey)
	})
	if err != nil {
		logrus.Error("iterate session error: ", err)
	}
	return sm
}

func (sm *PersistentSessionManager) Create(s *Session) error {
	err := sm.MemorySessionManger.Create(s)
	if err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		logrus.Warn(err)
		return nil
	}
	state.Put(fmt.Sprintf("/session/%s", s.ApiKey), b)
	return nil
}

func (sm *PersistentSessionManager) Expire(userID string) error {
	apiKeys := sm.MemorySessionManger.revSession[userID]
	logrus.Debug("expire api keys: ", apiKeys)
	for _, key := range apiKeys {
		err := state.Del(fmt.Sprintf("/session/%s", key))
		if err != nil {
			return err
		}
	}
	return sm.MemorySessionManger.Expire(userID)
}

func (sm *PersistentSessionManager) Delete(apiKey string) error {
	err := state.Del(fmt.Sprintf("/session/%s", apiKey))
	if err != nil {
		return err
	}
	return sm.MemorySessionManger.Delete(apiKey)
}
