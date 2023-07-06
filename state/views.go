package state

import (
	"context"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func updateViewCount(statusID string, count int) error {
	key := stateKey(fmt.Sprintf("/views/status/%s", statusID))
	resp, err := etcdClient.KV.Get(context.Background(), key)
	if err != nil {
		return err
	}

	if resp.Count == 0 {
		logrus.Debug("create views key ", key)
		r, err := etcdClient.Txn(context.Background()).
			If(clientv3.Compare(clientv3.Version(key), "=", 0)).
			Then(clientv3.OpPut(key, fmt.Sprintf("%d", count))).Commit()
		if err != nil {
			return err
		}
		if !r.Succeeded {
			return ErrTryAgainLater
		}
		return nil
	}

	logrus.Debug("update views key ", key)
	views, _ := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)

	r, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.ModRevision(key), "=", resp.Kvs[0].ModRevision)).
		Then(clientv3.OpPut(key, fmt.Sprintf("%d", count+int(views)))).Commit()
	if err != nil {
		return err
	}
	if !r.Succeeded {
		return ErrTryAgainLater
	}
	return nil
}
