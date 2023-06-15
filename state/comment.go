package state

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func StatusComments(statusID, after string, size int64) (ss []*Status) {
	statusCommentsKey := stateKey(fmt.Sprintf("/status/%s/comments/", statusID))

	ops := []clientv3.OpOption{
		clientv3.WithLimit(size),
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByCreateRevision, clientv3.SortDescend)}
	if len(after) > 0 {
		ops = append(ops, clientv3.WithRange(statusCommentsKey+after))
	}

	resp, err := etcdClient.KV.Get(context.Background(), statusCommentsKey, ops...)
	if err != nil {
		logrus.Debug(err)
		return
	}

	for _, kv := range resp.Kvs {
		resp, err = etcdClient.KV.Get(context.Background(), string(kv.Value))
		if err != nil {
			logrus.Debug(err)
			continue
		}
		if len(resp.Kvs) == 0 {
			logrus.Error("bad relation")
			continue
		}
		s := &Status{}
		err = json.Unmarshal(resp.Kvs[0].Value, s)
		if err != nil {
			logrus.Debug(err)
			continue
		}
		ss = append(ss, s)
	}
	return
}
