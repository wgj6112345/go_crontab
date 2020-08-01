package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"

	"github.com/wgj6112345/go-crontab/go_crontab/common"

	"github.com/coreos/etcd/mvcc/mvccpb"
)

var (
	G_jobMgr JobMgr
)

type JobMgr struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	watcher clientv3.Watcher
}

func InitJobMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
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

	watcher = clientv3.NewWatcher(client)
	G_jobMgr = JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	G_jobMgr.watchJobs()
	G_jobMgr.watchKill()
	fmt.Println("connect to etcd host: ", G_config.EtcdPoints)
	return
}

func (jobMgr *JobMgr) watchJobs() {
	var (
		getResp            *clientv3.GetResponse
		err                error
		cronJob            *common.CronJob
		watchChan          clientv3.WatchChan
		jobEvent           *common.JobEvent
		watchStartRevision int64
		jobName            string
	)
	if getResp, err = jobMgr.kv.Get(context.TODO(), common.CRON_JOB_DIR, clientv3.WithPrefix()); err != nil {
		fmt.Println(err)
	}

	if len(getResp.Kvs) != 0 {
		for _, kvpair := range getResp.Kvs {
			if cronJob, err = common.Unmarshal(kvpair.Value); err != nil {
				continue
			}

			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, cronJob)
			// 获取 job 发给 schedular
			G_schedular.PushJobEvent(jobEvent)

		}
	}

	// 开启 协程监听 此刻后的 key 变化
	go func() {
		watchStartRevision = getResp.Header.Revision + 1

		if watchChan = jobMgr.watcher.Watch(context.TODO(), common.CRON_JOB_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix()); err != nil {
			fmt.Println(err)
			return
		}

		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					// 更新事件
					fmt.Println("更新")
					if cronJob, err = common.Unmarshal(event.Kv.Value); err != nil {
						fmt.Println(err)
						continue
					}

					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, cronJob)
				case mvccpb.DELETE:
					// 删除事件
					// 获取 删除的任务名
					jobName = strings.TrimPrefix(string(event.Kv.Key), common.CRON_JOB_DIR)

					cronJob.Name = jobName
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, cronJob)
				}
				G_schedular.PushJobEvent(jobEvent)

			}
		}
	}()

	return
}

// watch kill 事件
func (jobMgr *JobMgr) watchKill() {
	var (
		err       error
		cronJob   *common.CronJob
		watchChan clientv3.WatchChan
		jobEvent  *common.JobEvent
		jobName   string
	)
	// 开启 协程监听  kill
	go func() {
		if watchChan = jobMgr.watcher.Watch(context.TODO(), common.CRON_KILL_JOB_DIR, clientv3.WithPrefix()); err != nil {
			fmt.Println(err)
			return
		}

		for watchResp := range watchChan {
			for _, event := range watchResp.Events {
				switch event.Type {
				case mvccpb.PUT:
					// 强杀事件  只需要获取 jobName 即可
					jobName = strings.TrimPrefix(string(event.Kv.Key), common.CRON_KILL_JOB_DIR)

					cronJob = &common.CronJob{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, cronJob)
				}
				G_schedular.PushJobEvent(jobEvent)
			}
		}
	}()

	return
}

func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *JobLock) {
	jobLock = InitJobLock(jobMgr.kv, jobMgr.lease, jobName)
	return
}
