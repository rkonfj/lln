package state

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rkonfj/lln/tools"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type StatusCommentsOptions struct {
	StatusID   string
	After      string
	Size       int64
	SortAscend bool
}

func StatusComments(statusID string, opts *tools.PaginationOptions) (ss []*Status, more bool) {
	statusCommentsKey := stateKey(fmt.Sprintf("/comments/status/%s/", statusID))
	return loadStatusByLinkerPagination(statusCommentsKey, opts)
}

func commentsCount(statusID string) int64 {
	return countKeys(stateKey(fmt.Sprintf("/comments/status/%s/", statusID)))
}

func viewCount(statusID string) int64 {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey(fmt.Sprintf("/views/status/%s", statusID)))
	if err != nil {
		logrus.Errorf("status %s view count etcd error: %s", statusID, err)
		return 0
	}
	if resp.Count == 0 {
		return 0
	}
	c, _ := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	return c
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
