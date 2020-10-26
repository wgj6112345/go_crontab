package master

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/wgj6112345/go-crontab/go_crontab/common"
)

var (
	G_jobMgr JobMgr
)

// 任务管理器
type JobMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

func InitJobMgr() (err error) {
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

	G_jobMgr = JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	fmt.Println("connect to etcd host: ", G_config.EtcdPoints)
	return
}

// 增删改查 job
// 增加任务
func (m *JobMgr) SaveJob(cronJob *common.CronJob) (oldJob common.CronJob, err error) {
	var (
		cronJobKey string
		bytes      []byte
		putResp    *clientv3.PutResponse
	)

	cronJobKey = common.CRON_JOB_DIR + cronJob.Name

	if bytes, err = json.Marshal(cronJob); err != nil {
		fmt.Println(err)
		return
	}

	if putResp, err = m.kv.Put(context.TODO(), cronJobKey, string(bytes), clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("job 保存成功, cronJobKey: ", cronJobKey, "string(bytes): ", string(bytes))

	// 尝试 get
	var getResp *clientv3.GetResponse
	if getResp, err = m.kv.Get(context.TODO(), cronJobKey); err != nil {
		fmt.Println("etcd get err: ", err)
		return
	}
	fmt.Println("cronJobKey: ", cronJobKey, string(getResp.Kvs[0].Value))

	// 如果是更新 ，返回旧值
	if putResp.PrevKv != nil {
		if err = json.Unmarshal(putResp.PrevKv.Value, &oldJob); err != nil {
			fmt.Println("json unmarshal err: ", err)
			err = nil //反序列化失败，可以忽略
			return
		}
	}

	return
}

// 删除任务
func (m *JobMgr) DeleteJob(name string) (oldJob common.CronJob, err error) {
	var (
		cronJobKey string
		delResp    *clientv3.DeleteResponse
	)

	cronJobKey = common.CRON_JOB_DIR + name

	if delResp, err = m.kv.Delete(context.TODO(), cronJobKey, clientv3.WithPrevKV()); err != nil {
		fmt.Println(err)
		return
	}

	// 如果是更新 ，返回旧值
	if delResp.PrevKvs != nil {
		if err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJob); err != nil {
			fmt.Println("err: ", err)
			err = nil //反序列化失败，可以忽略
			return
		}
	}

	return
}

func (m *JobMgr) ListJob() (jobs []*common.CronJob, err error) {
	var (
		cronJobKey string
		cronJob    *common.CronJob
		getResp    *clientv3.GetResponse
	)
	cronJobKey = common.CRON_JOB_DIR
	if getResp, err = m.kv.Get(context.TODO(), cronJobKey, clientv3.WithPrefix()); err != nil {
		fmt.Println(err)
	}

	for _, kvpair := range getResp.Kvs {
		cronJob = &common.CronJob{}
		if err = json.Unmarshal(kvpair.Value, cronJob); err != nil {
			continue
		}
		jobs = append(jobs, cronJob)
	}

	return
}

func (m *JobMgr) KillJob(name string) (err error) {
	var (
		cronKillKey    string
		leaseGrantResp *clientv3.LeaseGrantResponse
		leaseId        clientv3.LeaseID
	)

	cronKillKey = common.CRON_KILL_JOB_DIR + name

	if leaseGrantResp, err = m.lease.Grant(context.TODO(), 1); err != nil {
		fmt.Println(err)
		return
	}

	leaseId = leaseGrantResp.ID
	if _, err = m.kv.Put(context.TODO(), cronKillKey, "", clientv3.WithLease(leaseId)); err != nil {
		fmt.Println(err)
		return
	}
	return
}
