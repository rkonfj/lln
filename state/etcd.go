package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/pkg/transport"
)

var etcdClient *clientv3.Client

type EtcdOptions struct {
	Endpoints     []string
	CertFile      string
	KeyFile       string
	TrustedCAFile string
}

func InitState(opts EtcdOptions) (err error) {
	cfg := clientv3.Config{
		Endpoints:   opts.Endpoints,
		DialTimeout: 5 * time.Second,
	}

	tlsInfo := transport.TLSInfo{
		CertFile:      opts.CertFile,
		KeyFile:       opts.KeyFile,
		TrustedCAFile: opts.TrustedCAFile,
	}
	tlsConfig, err := tlsInfo.ClientConfig()
	if err != nil {
		logrus.Debug(err)
	} else {
		cfg.TLS = tlsConfig
	}

	etcdClient, err = clientv3.New(cfg)
	start()
	return
}

func stateKey(key string) string {
	if strings.HasPrefix(key, "/") {
		return fmt.Sprintf("/lln%s", key)
	}
	return fmt.Sprintf("/lln/%s", key)
}

func getPointerValue(key string) (*clientv3.GetResponse, error) {
	resp, err := etcdClient.KV.Get(context.Background(), key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) != 1 {
		return nil, fmt.Errorf("pointer %s not found", key)
	}
	return etcdClient.KV.Get(context.Background(), string(resp.Kvs[0].Value))
}

func Put(key string, value []byte) error {
	_, err := etcdClient.KV.Put(context.Background(), stateKey(key), string(value))
	return err
}

func Del(key string) error {
	_, err := etcdClient.KV.Delete(context.Background(), key)
	return err
}

func IterateWithPrefix(prefix string, handle func(key string, value []byte)) error {
	var lastCreateRev int64
	for {
		resp, err := etcdClient.KV.Get(context.Background(), stateKey(prefix),
			clientv3.WithPrefix(),
			clientv3.WithLimit(1024),
			clientv3.WithMinCreateRev(lastCreateRev+1))
		if err != nil {
			return err
		}
		for _, kv := range resp.Kvs {
			handle(string(kv.Key), kv.Value)
			lastCreateRev = kv.CreateRevision
		}
		if !resp.More {
			logrus.Debugf("[state] iterate prefix %s done", prefix)
			break
		}
	}
	return nil
}
