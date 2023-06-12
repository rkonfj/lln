package state

import (
	"context"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Label struct {
	Value string
	Count int64
}

func GetLabels() (labels []*Label) {
	resp, err := etcdClient.KV.Get(context.Background(), stateKey("/label/"),
		clientv3.WithSort(clientv3.SortByVersion, clientv3.SortDescend),
		clientv3.WithPrefix())
	if err != nil {
		logrus.Debug(err)
		return
	}
	for _, kv := range resp.Kvs {
		labels = append(labels, &Label{Value: string(kv.Value), Count: kv.Version})
	}
	return
}
