package state

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/decred/base58"
	"github.com/rkonfj/lln/tools"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	MsgTypeLike     string = "like"
	MsgTypeBookmark string = "bookmark"
	MsgTypeComment  string = "comment"
	MsgTypeAt       string = "at"
	MsgTypeFollow   string = "follow"
)

type Message struct {
	ID         string    `json:"id"`
	Message    string    `json:"message,omitempty"`
	From       *ActUser  `json:"from"`
	Type       string    `json:"type"`
	TargetID   string    `json:"targetID"`
	CreateTime time.Time `json:"createTime"`
}

func ListMessages(user *ActUser, opts *tools.PaginationOptions) (msgs []*Message, more bool) {
	ops := []clientv3.OpOption{
		clientv3.WithLimit(opts.Size),
		clientv3.WithPrefix(),
	}
	if opts.Ascend {
		ops = append(ops, clientv3.WithMinCreateRev(opts.After+1))
		ops = append(ops, clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortAscend))
	} else {
		if opts.After > 0 {
			ops = append(ops, clientv3.WithMaxCreateRev(opts.After-1))
		}
		ops = append(ops, clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortDescend))
	}
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf("/message/%s/", user.ID)), ops...)
	if err != nil {
		logrus.Error("ListMessages etcd error: ", err)
		return
	}
	more = resp.More
	for _, kv := range resp.Kvs {
		msg := &Message{}
		err = json.Unmarshal(kv.Value, msg)
		if err != nil {
			logrus.Error("ListMessages unmarshal error: ", err)
			continue
		}
		msgs = append(msgs, msg)
	}
	return
}

func DeleteMessages(user *ActUser, msgs []string) error {
	ops := []clientv3.Op{}
	for _, msgID := range msgs {
		key := stateKey(fmt.Sprintf("/message/%s/%s", user.ID, msgID))
		ops = append(ops, clientv3.OpDelete(key))
	}
	_, err := etcdClient.Txn(context.Background()).Then(ops...).Commit()
	return err
}

func ListTipMessages(user *ActUser, size int64) (msgs []string) {
	ops := []clientv3.OpOption{
		clientv3.WithLimit(size),
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortDescend)}
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf("/tips/message/%s/", user.ID)), ops...)
	if err != nil {
		logrus.Debug(err)
		return
	}
	linkerLen := len(stateKey(fmt.Sprintf("/message/%s/", user.ID)))
	for _, kv := range resp.Kvs {
		msgs = append(msgs, string(kv.Value)[linkerLen:])
	}
	return
}

func DeleteTipMessages(user *ActUser, msgs []string) error {
	ops := []clientv3.Op{}
	for _, msgID := range msgs {
		key := stateKey(fmt.Sprintf("/tips/message/%s/%s", user.ID, msgID))
		ops = append(ops, clientv3.OpDelete(key))
	}
	_, err := etcdClient.Txn(context.Background()).Then(ops...).Commit()
	return err
}

type MsgOptions struct {
	from     *ActUser
	toUID    string
	msgType  string
	targetID string
	message  string
}

func newMessageOps(opts MsgOptions) []clientv3.Op {
	msg := Message{
		ID:         base58.Encode(xid.New().Bytes()),
		From:       opts.from,
		Type:       opts.msgType,
		Message:    opts.message,
		TargetID:   opts.targetID,
		CreateTime: time.Now(),
	}
	msgB, _ := json.Marshal(msg)
	msgKey := stateKey(fmt.Sprintf("/message/%s/%s", opts.toUID, msg.ID))
	msgNewKey := stateKey(fmt.Sprintf("/tips/message/%s/%s", opts.toUID, msg.ID))

	return []clientv3.Op{
		clientv3.OpPut(msgKey, string(msgB)),
		clientv3.OpPut(msgNewKey, msgKey),
	}
}
