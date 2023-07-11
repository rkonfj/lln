package state

import (
	"context"
	"encoding/json"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Settings struct {
	TermsOfService string   `json:"termsOfService"`
	PrivacyPolicy  string   `json:"privacyPolicy"`
	Friends        []Friend `json:"friends"`
	Announcement   string   `json:"announcement"`
	ModRev         int64    `json:"modRev"`
}

type Friend struct {
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Logo  string `json:"logo"`
	URL   string `json:"url"`
}

func GetSettings() (s *Settings, err error) {
	key := stateKey("/settings")
	resp, err := etcdClient.KV.Get(context.Background(), key)
	if err != nil {
		return
	}
	if resp.Count == 0 {
		s = &Settings{
			TermsOfService: "/termsofservice",
			PrivacyPolicy:  "/privacypolicy",
			Friends:        []Friend{},
		}
		b, _ := json.Marshal(s)
		r, er := etcdClient.Txn(context.Background()).
			If(clientv3.Compare(clientv3.Version(key), "=", 0)).
			Then(clientv3.OpPut(key, string(b))).Commit()
		if er != nil {
			err = er
			return
		}
		if !r.Succeeded {
			err = ErrTryAgainLater
		}
		return
	}
	s = &Settings{}
	err = json.Unmarshal(resp.Kvs[0].Value, s)
	s.ModRev = resp.Kvs[0].ModRevision
	return
}

func UpdateSettings(s *Settings) error {
	key := stateKey("/settings")
	resp, err := etcdClient.KV.Get(context.Background(), key)
	if err != nil {
		return err
	}

	if resp.Count == 0 {
		return ErrTryAgainLater
	}

	b, _ := json.Marshal(s)

	r, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", resp.Kvs[0].ModRevision)).
		Then(clientv3.OpPut(key, string(b))).Commit()
	if err != nil {
		return err
	}
	if !r.Succeeded {
		return ErrTryAgainLater
	}
	return nil
}
