package state

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

var (
	UserChanged chan *User             = make(chan *User, 128)
	SVCM        StatusViewCountManager = StatusViewCountManager{
		ctx: make(map[string]int),
	}
	lastStatusCreateRev string = stateKey("/recommended/lastrev")
)

type StatusViewCountManager struct {
	ctx          map[string]int
	pendingCount int
	l            sync.RWMutex
}

func (m *StatusViewCountManager) Viewed(statusID string) {
	m.l.Lock()
	defer m.l.Unlock()
	m.ctx[statusID] = m.ctx[statusID] + 1
	m.pendingCount++
}

func startKeepConsistency() {
	go keepStatusUserConsistentLoop()
	go keepStatusViewCountConsistentLoop()
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

func keepStatusViewCountConsistentLoop() {
	for {
		time.Sleep(10 * time.Second)
		SVCM.l.Lock()
		if SVCM.pendingCount > 0 {
			for k, v := range SVCM.ctx {
				for i := 0; i < 10; i++ {
					err := updateViewCount(k, v)
					if err == nil {
						break
					}
					logrus.Warn("update views count error: ", err)
					time.Sleep(100 * time.Millisecond)
				}
				delete(SVCM.ctx, k)
			}
			SVCM.pendingCount = 0
		}
		SVCM.l.Unlock()
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
	// 评论推荐
	if len(s.RefStatus) > 0 {
		return recommandComment(s, del)
	}

	// 用户头像为空不推荐
	if len(s.User.Picture) == 0 {
		return nil
	}

	if del {
		delKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.ID))
		_, err := etcdClient.KV.Delete(context.Background(), delKey)
		if err != nil {
			logrus.Error(err)
		}
		return nil
	}

	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	putKey := stateKey(fmt.Sprintf("/recommended/status/%s", s.ID))
	ops := []clientv3.Op{clientv3.OpPut(putKey, statusKey),
		clientv3.OpPut(lastStatusCreateRev, fmt.Sprintf("%d", s.CreateRev))}
	_, err := etcdClient.Txn(context.Background()).
		Then(ops...).Commit()
	return err
}

func recommandComment(s *Status, del bool) error {
	meta, err := NewCommentsRecommandMetaForRecommand(s.RefStatus)
	if err != nil {
		return fmt.Errorf("[recommended-algo] %s comments meta error: %s", s.RefStatus, err)
	}

	for i := 0; i < 10; i++ {
		err = meta.RunRecommand(s.CreateRev)
		if err != nil {
			logrus.Warnf("[recommended-algo] retry %s comments recommand: %s", s.RefStatus, err)
			time.Sleep(time.Second)
			continue
		}
		return nil
	}
	return fmt.Errorf("[recommended-algo] comments recommand error: 30 times retry, finally failed")
}
