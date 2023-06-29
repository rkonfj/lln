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
	Content   []*StatusFragment
	RefStatus string
	User      *ActUser
	Labels    []string
	At        []string
}

type Status struct {
	ID         string            `json:"id"`
	Content    []*StatusFragment `json:"content"`
	RefStatus  string            `json:"prev"`
	User       *ActUser          `json:"user"`
	CreateRev  int64             `json:"createRev"`
	CreateTime time.Time         `json:"createTime"`
	Comments   int64             `json:"comments"`
	LikeCount  int64             `json:"likeCount"`
	Views      int64             `json:"views"`
	Bookmarks  int64             `json:"bookmarks"`
}

type StatusFragment struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

func (s *Status) Overview() string {
	for _, c := range s.Content {
		if c.Type == "text" {
			return c.Value
		}
	}
	return ""
}

func NewStatus(opts *StatusOptions) (*Status, error) {
	s := &Status{
		ID:         base58.Encode(xid.New().Bytes()),
		Content:    opts.Content,
		RefStatus:  opts.RefStatus,
		User:       opts.User,
		CreateTime: time.Now(),
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	txn := etcdClient.Txn(context.Background())
	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	userStatusKey := stateKey(fmt.Sprintf("/%s/status/%s", s.User.ID, s.ID))
	statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s/%s", s.RefStatus, s.ID))
	ops := []clientv3.Op{
		clientv3.OpPut(statusKey, string(b)),
		clientv3.OpPut(userStatusKey, statusKey),
	}

	if len(s.RefStatus) > 0 {
		ops = append(ops, clientv3.OpPut(statusCommentsKey, statusKey))
		s := GetStatus(s.RefStatus)
		if s != nil {
			ops = append(ops, newMessageOps(MsgOptions{
				from:     opts.User,
				toUID:    s.User.ID,
				msgType:  MsgTypeComment,
				targetID: s.ID,
				message:  s.Overview(),
			})...)
		}
	}

	if len(opts.At) > 0 {
		for _, at := range util.Unique(opts.At) {
			u := UserByUniqueName(at)
			if u != nil {
				ops = append(ops, newMessageOps(MsgOptions{
					from:     opts.User,
					toUID:    u.ID,
					msgType:  MsgTypeAt,
					targetID: s.ID,
					message:  s.Overview(),
				})...)
			}
		}
	}

	if len(opts.Labels) > 0 {
		for _, l := range util.Unique(opts.Labels) {
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
	statusKey := stateKey(fmt.Sprintf("/status/%s", statusID))
	resp, err := etcdClient.KV.Get(context.Background(), statusKey)
	if err != nil {
		logrus.Debug(err)
		return nil
	}

	if len(resp.Kvs) == 0 {
		logrus.Debugf("status %s not found", statusKey)
		return nil

	}

	s, err := unmarshalStatus(resp.Kvs[0].Value, resp.Kvs[0].CreateRevision)
	if err != nil {
		logrus.Debug(err)
		return nil
	}
	return s
}

func UserStatus(uniqueName, after string, size int64) []*Status {
	return UserByUniqueName(uniqueName).ListStatus(after, size)
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
		s, err := unmarshalStatus(r.Kvs[0].Value, r.Kvs[0].CreateRevision)
		if err != nil {
			logrus.Error(err)
			continue
		}
		ss = append(ss, s)
	}
	return ss
}

func Recommendations(user *ActUser, after string, size int64) (ss []*Status) {
	return loadStatusByLinker(stateKey("/recommended/status/"), after, size)
}

func ListStatusByLabel(value, after string, size int64) []*Status {
	return loadStatusByLinker(stateKey(fmt.Sprintf("/labels/%s/status/", value)), after, size)
}

func ListStatusByKeyword(value, after string, size int64) []*Status {
	return nil
}

func unmarshalStatus(b []byte, cRev int64) (s *Status, err error) {
	s = &Status{}
	err = json.Unmarshal(b, s)
	if err != nil {
		return nil, err
	}
	s.CreateRev = cRev
	s.Comments = commentsCount(s.ID)
	s.LikeCount = likeCount(s.ID)
	s.Views = viewCount(s.ID)
	s.Bookmarks = bookmarkCount(s.ID)
	return s, nil
}
