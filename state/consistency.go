package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var (
	UserChanged   chan *User = make(chan *User, 128)
	lastStatuskey string     = stateKey("/recommended/last")
)

func start() {
	go keepStatusUserConsistentLoop()
	go keepRecommendedStatusLoop()
}

func keepStatusUserConsistentLoop() {
	for e := range UserChanged {
		lastKey := ""
		for {
			ss := e.ListStatus(lastKey, 20)
			if ss == nil {
				break
			}
			for i, s := range ss {
				if i == len(ss)-1 {
					lastKey = s.ID
				}
				if s.User.Name != e.Name || s.User.UniqueName != e.UniqueName {
					key := stateKey(fmt.Sprintf("/status/%s", s.ID))
					s.User.Name = e.Name
					s.User.UniqueName = e.UniqueName
					b, err := json.Marshal(s)
					if err != nil {
						logrus.Debug(err)
						continue
					}
					_, err = etcdClient.Put(context.Background(), key, string(b))
					if err != nil {
						logrus.Error(err)
					}
				}
			}
		}
	}
}

func keepRecommendedStatusLoop() {
	session, err := concurrency.NewSession(etcdClient)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer session.Close()
	mutex := concurrency.NewMutex(session, stateKey("/election/recommended"))

	if err := mutex.Lock(context.Background()); err != nil {
		logrus.Error(err)
		return
	}

	logrus.Info("keepRecommendedStatusLoop act as leader")

	lastStatus, err := etcdClient.KV.Get(context.Background(), lastStatuskey, clientv3.WithCountOnly())
	if err != nil {
		logrus.Error(err)
		return
	}

	opts := []clientv3.OpOption{
		clientv3.WithLimit(1024),
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortAscend)}
	if len(lastStatus.Kvs) > 0 {
		opts = append(opts, clientv3.WithRange(stateKey(fmt.Sprintf("/status/%s", lastStatus.Kvs[0].Value))))
	}
	resp, err := etcdClient.KV.Get(context.Background(), stateKey("/status/"), opts...)
	if err != nil {
		logrus.Error(err)
		return
	}

	for _, kv := range resp.Kvs {
		s, err := unmarshalStatus(kv.Value)
		if err != nil {
			logrus.Error(err)
			continue
		}
		err = recommend(s)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}

	logrus.Infof("keepRecommendedStatusLoop process %d status successfully", resp.Count)

	rch := etcdClient.Watch(context.Background(), stateKey("/status/"), clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if ev.IsCreate() {
				s, err := unmarshalStatus(ev.Kv.Value)
				if err != nil {
					logrus.Error(err)
					continue
				}
				err = recommend(s)
				if err != nil {
					logrus.Error(err)
					continue
				}
			}
		}
	}
}

func recommend(s *Status) error {
	if len(s.User.Picture) == 0 {
		return nil
	}
	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	putKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.ID))
	if len(s.RefStatus) > 0 {
		delKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.RefStatus))
		_, err := etcdClient.Txn(context.Background()).
			If(clientv3.Compare(clientv3.Version(delKey), "=", 0)).
			Then(clientv3.OpPut(putKey, statusKey), clientv3.OpPut(lastStatuskey, s.ID)).
			Else(clientv3.OpPut(putKey, statusKey), clientv3.OpPut(lastStatuskey, s.ID),
				clientv3.OpDelete(delKey)).Commit()
		return err
	}
	_, err := etcdClient.Txn(context.Background()).
		Then(clientv3.OpPut(putKey, statusKey), clientv3.OpPut(lastStatuskey, s.ID)).Commit()
	return err
}
