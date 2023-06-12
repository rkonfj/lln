package state

import (
	"context"
	"errors"
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
		return nil, errors.New("not found")
	}
	return etcdClient.KV.Get(context.Background(), string(resp.Kvs[0].Value))
}
