package master

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/mvcc/mvccpb"

	"github.com/coreos/etcd/clientv3"
	"github.com/wgj6112345/go-crontab/go_crontab/common"
)

var (
	G_workerMgr WorkerMgr
)

type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

func InitWorkerMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
	)

	config = clientv3.Config{
		Endpoints:   G_config.EtcdPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	if client, err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	G_workerMgr = WorkerMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return

}

func (workerMgr *WorkerMgr) ListWorkers() (workerList []string, err error) {
	var (
		getResp   *clientv3.GetResponse
		kv        *mvccpb.KeyValue
		workerKey string
		workerIp  string
	)

	if getResp, err = workerMgr.kv.Get(context.TODO(), common.CRON_WORKER_DIR, clientv3.WithPrefix()); err != nil {
		fmt.Println(err)
		return
	}

	for _, kv = range getResp.Kvs {
		workerKey = string(kv.Key)
		workerIp = extraceWorkerIp(workerKey)
		workerList = append(workerList, workerIp)
	}

	return
}

func extraceWorkerIp(workerKey string) (workerIp string) {
	workerIp = strings.TrimPrefix(workerKey, common.CRON_WORKER_DIR)
	return
}
