package state

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func StatusComments(statusID, after string, size int64) (ss []*Status) {
	statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s/", statusID))

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
		s, err := unmarshalStatus(resp.Kvs[0].Value)
		if err != nil {
			logrus.Debug(err)
			continue
		}
		ss = append(ss, s)
	}
	return
}

func commentsCount(statusID string) int64 {
	return countKeys(stateKey(fmt.Sprintf("/comments/status/%s/", statusID)))
}

func likeCount(statusID string) int64 {
	return countKeys(stateKey(fmt.Sprintf("/like/status/%s/", statusID)))
}

func viewCount(statusID string) int64 {
	return countKeys(stateKey(fmt.Sprintf("/view/status/%s/", statusID)))
}

func countKeys(key string) int64 {
	resp, err := etcdClient.KV.Get(context.Background(), key,
		clientv3.WithCountOnly(), clientv3.WithPrefix())
	if err != nil {
		logrus.Debug(err)
		return -1
	}
	return resp.Count
}
