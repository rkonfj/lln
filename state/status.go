package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/tools"
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
	Disabled   bool              `json:"disabled"`
}

type StatusFragment struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

func (s *Status) Overview() string {
	for _, c := range s.Content {
		if c.Type == "text" {
			return strings.TrimSpace(strings.ReplaceAll(c.Value, "\n", ""))
		}
	}
	return ""
}

func (s *Status) ContentsByType(t string) (sfs []*StatusFragment) {
	for _, sf := range s.Content {
		if sf.Type == t {
			sfs = append(sfs, sf)
		}
	}
	return
}

func (s *Status) Delete(uid string) error {
	if s.User.ID != uid {
		return ErrStatusNotFound
	}
	statusProbeKey := stateKey(fmt.Sprintf("/probe/status/%s", s.ID))
	resp, err := etcdClient.KV.Get(context.Background(), statusProbeKey)
	if err != nil {
		return err
	}
	cmps := []clientv3.Cmp{}
	if resp.Count > 0 {
		// disable delete when comments count greater than 0
		if string(resp.Kvs[0].Value) != s.ID {
			statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s", s.ID))
			r, err := etcdClient.Get(context.Background(), statusCommentsKey,
				clientv3.WithCountOnly(), clientv3.WithPrefix())
			if err != nil || r.Count != 0 {
				logrus.Error("", err)
				return errors.New("there are quotes")
			}
		}
		// there are no new comments when executing txn
		cmps = append(cmps, clientv3.Compare(
			clientv3.ModRevision(statusProbeKey), "=", resp.Kvs[0].ModRevision))
	}

	statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s/%s", s.RefStatus, s.ID))
	statusRecycleKey := stateKey(fmt.Sprintf("/recycle/status/%s", s.ID))
	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	userStatusKey := stateKey(fmt.Sprintf("/%s/status/%s", uid, s.ID))

	b, _ := json.Marshal(s)

	txnResp, err := etcdClient.Txn(context.Background()).If(cmps...).
		Then(clientv3.OpDelete(statusKey),
			clientv3.OpDelete(userStatusKey),
			clientv3.OpDelete(statusProbeKey),
			clientv3.OpDelete(statusCommentsKey),
			clientv3.OpPut(statusRecycleKey, string(b))).Commit()
	if err != nil {
		return err
	}

	if !txnResp.Succeeded {
		// disable delete when comments count greater than 0
		return ErrStatusQuotes
	}
	return nil
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

	statusKey := stateKey(fmt.Sprintf("/status/%s", s.ID))
	userStatusKey := stateKey(fmt.Sprintf("/%s/status/%s", s.User.ID, s.ID))
	statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s/%s", s.RefStatus, s.ID))
	statusProbeKey := stateKey(fmt.Sprintf("/probe/status/%s", s.ID))
	ops := []clientv3.Op{
		clientv3.OpPut(statusKey, string(b)),
		clientv3.OpPut(userStatusKey, statusKey),
		clientv3.OpPut(statusProbeKey, s.ID),
	}

	cmps := []clientv3.Cmp{}

	if len(s.RefStatus) > 0 {
		refProbeKey := stateKey(fmt.Sprintf("/probe/status/%s", s.RefStatus))
		cmps = append(cmps, clientv3.Compare(clientv3.Version(refProbeKey), "!=", 0))
		ops = append(ops, clientv3.OpPut(statusCommentsKey, statusKey))
		ops = append(ops, clientv3.OpPut(refProbeKey, s.ID))
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
		for _, at := range tools.Unique(opts.At) {
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
		for _, l := range tools.Unique(opts.Labels) {
			key := stateKey(fmt.Sprintf("/labels/%s/status/%s", l, s.ID))
			ops = append(ops, clientv3.OpPut(key, statusKey))
			key = stateKey(fmt.Sprintf("/label/%s", l))
			ops = append(ops, clientv3.OpPut(key, l))
		}
	}

	_, err = etcdClient.Txn(context.Background()).If(cmps...).Then(ops...).Commit()

	if err != nil {
		return nil, err
	}
	return s, nil
}

func RecommendStatus(statusID string) error {
	key := stateKey(fmt.Sprintf("/recommended/status/%s", statusID))
	statusKey := stateKey(fmt.Sprintf("/status/%s", statusID))
	_, err := etcdClient.KV.Put(context.Background(), key, statusKey)
	return err
}

func NotRecommendStatus(statusID string) error {
	key := stateKey(fmt.Sprintf("/recommended/status/%s", statusID))
	_, err := etcdClient.KV.Delete(context.Background(), key)
	return err
}

func getStatusBin(statusID string) (s []byte, createRev int64) {
	statusKey := stateKey(fmt.Sprintf("/status/%s", statusID))
	resp, err := etcdClient.KV.Get(context.Background(), statusKey)
	if err != nil {
		logrus.Debug(err)
		return
	}

	if len(resp.Kvs) == 0 {
		logrus.Debugf("status %s not found", statusKey)
		return

	}
	return resp.Kvs[0].Value, resp.Kvs[0].CreateRevision

}

func GetStatus(statusID string) *Status {
	s, err := unmarshalStatus(getStatusBin(statusID))
	if err != nil {
		logrus.Debug(err)
		return nil
	}
	return s
}

func loadStatusByLinkerPagination(prefixKey string, options *tools.PaginationOptions) (ss []*Status, more bool) {
	if options == nil {
		options = &tools.PaginationOptions{}
	}
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithLimit(options.Size),
	}
	if options.Ascend {
		opts = append(opts, clientv3.WithMinCreateRev(options.After+1))
		opts = append(opts, clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortAscend))
	} else {
		if options.After > 0 {
			opts = append(opts, clientv3.WithMaxCreateRev(options.After-1))
		}
		opts = append(opts, clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortDescend))
	}
	resp, err := etcdClient.KV.Get(context.Background(), prefixKey, opts...)
	if err != nil {
		logrus.Errorf("prefix %s pagination etcd error: %s", prefixKey, err)
		return
	}
	for _, kv := range resp.Kvs {
		r, err := etcdClient.Get(context.Background(), string(kv.Value))
		if err != nil {
			logrus.Error(err)
			continue
		}
		if len(r.Kvs) == 0 {
			logrus.Errorf("not found %s -> %s ", string(kv.Key), string(kv.Value))
			continue
		}
		s, err := unmarshalStatus(r.Kvs[0].Value, kv.CreateRevision)
		if err != nil {
			logrus.Error(err)
			continue
		}
		ss = append(ss, s)
	}
	return ss, resp.More
}

func Recommendations(user *ActUser, opts *tools.PaginationOptions) (ss []*Status, more bool) {
	return loadStatusByLinkerPagination(stateKey("/recommended/status/"), opts)
}

func RecommendCount(user *ActUser, createRev int64) int {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey("/recommended/status/"),
		clientv3.WithPrefix(), clientv3.WithLimit(128), clientv3.WithMinCreateRev(createRev+1))
	if err != nil {
		logrus.Error("RecommendCount etcd error: ", err)
		return 0
	}
	return len(resp.Kvs)
}

func ListStatusByLabel(value string, opts *tools.PaginationOptions) ([]*Status, bool) {
	return loadStatusByLinkerPagination(stateKey(fmt.Sprintf("/labels/%s/status/", value)), opts)
}

func ListStatusByKeyword(value string, opts *tools.PaginationOptions) []*Status {
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
	s.Disabled = statusDisabled(s.ID)
	return s, nil
}

func statusDisabled(statusID string) bool {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf("/probe/status/%s", statusID)))
	if err != nil {
		logrus.Error("statusDisabled etcd error:", err.Error())
		return false
	}
	return resp.Count == 0
}
