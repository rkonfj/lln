package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/util"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type StatusOptions struct {
	Content   []StatusFragment
	RefStatus string
	User      *ActUser
	Labels    []string
}

type Status struct {
	ID         string           `json:"id"`
	Content    []StatusFragment `json:"content"`
	RefStatus  string           `json:"prev"`
	User       *ActUser         `json:"user"`
	CreateTime time.Time        `json:"createTime"`
	Labels     []string         `json:"labels"`
	Comments   int64            `json:"comments"`
	LikeCount  int64            `json:"likeCount"`
	Views      int64            `json:"views"`
}

type StatusFragment struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

func NewStatus(opts *StatusOptions) (*Status, error) {
	s := &Status{
		ID:         base58.Encode(xid.New().Bytes()),
		Content:    opts.Content,
		RefStatus:  opts.RefStatus,
		User:       opts.User,
		CreateTime: time.Now(),
		Labels:     opts.Labels,
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	txn := etcdClient.Txn(context.Background())
	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	userStatusKey := stateKey(fmt.Sprintf("/%s/status/%s", s.User.ID, s.ID))
	userUniqueNameKey := stateKey(fmt.Sprintf("/%s/status/%s", s.User.UniqueName, s.ID))
	statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s/%s", s.RefStatus, s.ID))
	ops := []clientv3.Op{
		clientv3.OpPut(statusKey, string(b)),
		clientv3.OpPut(userStatusKey, statusKey),
	}

	if userStatusKey != userUniqueNameKey {
		ops = append(ops, clientv3.OpPut(userUniqueNameKey, statusKey))
	}

	if len(s.RefStatus) > 0 {
		ops = append(ops, clientv3.OpPut(statusCommentsKey, statusKey))
	}

	if len(s.Labels) > 0 {
		for _, l := range util.Unique(s.Labels) {
			key := stateKey(fmt.Sprintf("/labels/%s/status/%s", l, s.ID))
			ops = append(ops, clientv3.OpPut(key, statusKey))
			key = stateKey(fmt.Sprintf("/label/%s", l))
			ops = append(ops, clientv3.OpPut(key, l))
		}
	}

	_, err = txn.Then(ops...).Commit()

	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetStatus(statusID string) *Status {
	statusLinkerKey := stateKey(fmt.Sprintf("/status/%s", statusID))
	resp, err := etcdClient.KV.Get(context.Background(), statusLinkerKey)
	if err != nil {
		logrus.Debug(err)
		return nil
	}

	if len(resp.Kvs) == 0 {
		logrus.Debug("not found")
		return nil
	}

	s, err := unmarshalStatus(resp.Kvs[0].Value)
	if err != nil {
		logrus.Debug(err)
		return nil
	}
	return s
}

func UserStatus(uniqueName, after string, size int64) (ss []*Status) {
	return loadStatusByLinker(stateKey(fmt.Sprintf("/%s/status", uniqueName)), after, size)
}

func loadStatusByLinker(key, after string, size int64) (ss []*Status) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithLimit(size),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend)}
	if len(after) > 0 {
		opts = append(opts, clientv3.WithRange(key+after))
	}
	resp, err := etcdClient.KV.Get(context.Background(), key, opts...)
	if err != nil {
		logrus.Debug(err)
		return
	}
	for _, kv := range resp.Kvs {
		r, err := etcdClient.Get(context.Background(), string(kv.Value))
		if err != nil {
			logrus.Error(err)
			continue
		}
		if len(r.Kvs) == 0 {
			logrus.Error("not found ", string(kv.Value))
			continue
		}
		s, err := unmarshalStatus(r.Kvs[0].Value)
		if err != nil {
			logrus.Error(err)
			continue
		}
		ss = append(ss, s)
	}
	return ss
}

func LikeStatus(user *ActUser, statusID string) error {
	statusLikeSetKey := stateKey(fmt.Sprintf("/like/status/%s/%s", statusID, user.ID))
	b, err := json.Marshal(user)
	if err != nil {
		return err
	}
	_, err = etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Version(statusLikeSetKey), ">", 0)).
		Then(clientv3.OpDelete(statusLikeSetKey)).
		Else(clientv3.OpPut(statusLikeSetKey, string(b))).
		Commit()
	return err
}

func Recommendations(user *ActUser, after string, size int64) (ss []*Status) {
	ops := []clientv3.OpOption{
		clientv3.WithLimit(size),
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortDescend)}
	if len(after) > 0 {
		ops = append(ops, clientv3.WithRange(stateKey(fmt.Sprintf("/status/%s", after))))
	}
	resp, err := etcdClient.KV.Get(context.Background(), stateKey("/status"), ops...)
	if err != nil {
		logrus.Debug(err)
		return
	}
	for _, kv := range resp.Kvs {
		s, err := unmarshalStatus(kv.Value)
		if err != nil {
			logrus.Debug(err)
			continue
		}
		ss = append(ss, s)
	}
	return
}

func unmarshalStatus(b []byte) (s *Status, err error) {
	s = &Status{}
	err = json.Unmarshal(b, s)
	if err != nil {
		return nil, err
	}
	s.Comments = commentsCount(s.ID)
	s.LikeCount = likeCount(s.ID)
	s.Views = viewCount(s.ID)
	return s, nil
}
