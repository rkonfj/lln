package state

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var (
	UserChanged         chan *User = make(chan *User, 128)
	lastStatusCreateRev string     = stateKey("/recommended/lastrev")
)

func startKeepConsistency() {
	go keepStatusUserConsistentLoop()
	go keepSessionConsistentLoop()
	go keepRecommendedStatusLoop()
}

func keepStatusUserConsistentLoop() {
	for e := range UserChanged {
		lastRev := int64(0)
		for {
			ss, more := e.ListStatus(&tools.PaginationOptions{After: lastRev, Size: 20})
			if ss == nil {
				break
			}
			for i, s := range ss {
				if i == len(ss)-1 {
					lastRev = s.CreateRev
				}
				if s.User.Name != e.Name ||
					s.User.UniqueName != e.UniqueName ||
					s.User.VerifiedCode != e.VerifiedCode {
					key := stateKey(fmt.Sprintf("/status/%s", s.ID))
					s.User.Name = e.Name
					s.User.UniqueName = e.UniqueName
					s.User.VerifiedCode = e.VerifiedCode
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
			if !more {
				break
			}
		}
	}
}

func keepSessionConsistentLoop() {
	rch := etcdClient.Watch(context.Background(), stateKey("/session/"),
		clientv3.WithPrefix(), clientv3.WithPrevKV())
	sm := DefaultSessionManager.(*PersistentSessionManager)
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if ev.IsCreate() {
				s := Session{}
				err := json.Unmarshal(ev.Kv.Value, &s)
				if err != nil {
					logrus.Error("[session create] invalid session struct: ", err)
					continue
				}
				err = sm.MemorySessionManger.Create(&s)
				if err != nil {
					logrus.Error("[session create] create error: ", err)
				}
				logrus.Debug("[session create] synced session ", s.ApiKey)
			}

			if ev.Type == clientv3.EventTypeDelete {
				s := Session{}
				err := json.Unmarshal(ev.PrevKv.Value, &s)
				if err != nil {
					logrus.Error("[session delete] invalid session struct: ", err)
					continue
				}
				err = sm.MemorySessionManger.Delete(s.ApiKey)
				if err != nil {
					logrus.Error("[session delete] delete error: ", err)
				}
				logrus.Debug("[session delete] removed session ", s.ApiKey)
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

	logrus.Info("[recommended-algo] act as leader")

	lastCreateRev, err := etcdClient.KV.Get(context.Background(), lastStatusCreateRev)
	if err != nil {
		logrus.Error(err)
		return
	}

	createRev := int64(0)

	if lastCreateRev.Count > 0 {
		createRev, err = strconv.ParseInt(string(lastCreateRev.Kvs[0].Value), 10, 64)
		if err != nil {
			logrus.Error(err)
			return
		}
	}

	opts := []clientv3.OpOption{
		clientv3.WithLimit(1024),
		clientv3.WithPrefix(),
		clientv3.WithMinCreateRev(createRev + 1),
		clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortAscend)}
	resp, err := etcdClient.KV.Get(context.Background(), stateKey("/status/"), opts...)
	if err != nil {
		logrus.Error(err)
		return
	}

	for _, kv := range resp.Kvs {
		s, err := unmarshalStatus(kv.Value, kv.CreateRevision)
		if err != nil {
			logrus.Error(err)
			continue
		}
		err = recommend(s, false)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}

	if len(resp.Kvs) > 0 {
		logrus.Infof("[recommended-algo] process %d status successfully", len(resp.Kvs))
	} else {
		logrus.Infof("[recommended-algo] everything is ok")
	}

	rch := etcdClient.Watch(context.Background(), stateKey("/status/"),
		clientv3.WithPrefix(), clientv3.WithPrevKV())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if ev.IsCreate() || ev.Type == clientv3.EventTypeDelete {
				del := ev.Type == clientv3.EventTypeDelete
				kv := ev.Kv
				if del {
					kv = ev.PrevKv
				}
				s, err := unmarshalStatus(kv.Value, kv.CreateRevision)
				if err != nil {
					logrus.Error(err)
					continue
				}
				logrus.Debugf("[recommended-algo] apply recommend algo to %s", ev.Kv.Key)
				err = recommend(s, del)
				if err != nil {
					logrus.Error(err)
				}
				continue
			}
		}
	}
}

func recommend(s *Status, del bool) error {
	if len(s.User.Picture) == 0 {
		return nil
	}

	if del {
		delKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.ID))
		_, err := etcdClient.KV.Delete(context.Background(), delKey)
		if err != nil {
			logrus.Error(err)
		}

		if len(s.RefStatus) > 0 {
			statusKey := stateKey(fmt.Sprintf("/status/%s", s.RefStatus))
			putKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.RefStatus))
			_, err = etcdClient.KV.Put(context.Background(), putKey, statusKey)
			if err != nil {
				logrus.Error(err)
			}
		}
		return nil
	}

	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	putKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.ID))
	ops := []clientv3.Op{clientv3.OpPut(putKey, statusKey),
		clientv3.OpPut(lastStatusCreateRev, fmt.Sprintf("%d", s.CreateRev))}
	if len(s.RefStatus) > 0 {
		delKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.RefStatus))
		_, err := etcdClient.Txn(context.Background()).
			If(clientv3.Compare(clientv3.Version(delKey), "=", 0)).
			Then(ops...).
			Else(append(ops, clientv3.OpDelete(delKey))...).Commit()
		return err
	}
	_, err := etcdClient.Txn(context.Background()).
		Then(ops...).Commit()
	return err
}
