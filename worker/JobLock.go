package worker

import (
	"context"
	"fmt"

	"github.com/wgj6112345/go-crontab/go_crontab/common"

	"github.com/coreos/etcd/clientv3"
)

type JobLock struct {
	kv clientv3.KV

	lease      clientv3.Lease
	leaseId    clientv3.LeaseID   // 用于释放锁
	cancelFunc context.CancelFunc // 用于释放锁

	jobName  string
	isLocked bool
}

func InitJobLock(kv clientv3.KV, lease clientv3.Lease, jobName string) *JobLock {
	return &JobLock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
}

// 尝试获取锁
func (jobLock *JobLock) TryLock() (err error) {
	var (
		txn                clientv3.Txn
		jobLockKey         string
		ctx                context.Context
		cancelFunc         context.CancelFunc
		leaseGrantResponse *clientv3.LeaseGrantResponse
		leaseId            clientv3.LeaseID
		leaseKeepChan      <-chan *clientv3.LeaseKeepAliveResponse
		leaseKeep          *clientv3.LeaseKeepAliveResponse
		txnResp            *clientv3.TxnResponse
	)
	// 创建 ectd 分布式锁：1.创建租约；2.自动续租；3.开启事务，创建锁；4.关闭自动续租，租约设置为 1 秒
	// 创建租约
	if leaseGrantResponse, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}
	leaseId = leaseGrantResponse.ID

	// 开启协程自动续租
	ctx, cancelFunc = context.WithCancel(context.TODO())
	if leaseKeepChan, err = jobLock.lease.KeepAlive(ctx, leaseId); err != nil {
		goto FAIL
	}

	go func() {
		for {
			select {
			case leaseKeep = <-leaseKeepChan:
				if leaseKeep == nil {
					goto END
				}
			}
		}
	END:
	}()

	// 开启事务
	jobLockKey = common.CRON_JOB_LOCK_DIR + jobLock.jobName
	txn = jobLock.kv.Txn(context.TODO())
	txn.If(clientv3.Compare(clientv3.CreateRevision(jobLockKey), "=", 0)).
		Then(clientv3.OpPut(jobLockKey, "", clientv3.WithLease(leaseId))). // optput 中  withlease 用于释放锁
		Else(clientv3.OpGet(jobLockKey))

	// 提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}

	if !txnResp.Succeeded {
		fmt.Printf("锁：%v 已被占用\n", jobLockKey)
		err = common.ERR_JOB_LOCK_ALREADY_USED
		goto FAIL
	}

	jobLock.leaseId = leaseId
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true

	return
FAIL:
	cancelFunc()
	jobLock.lease.Revoke(context.TODO(), leaseId)
	return
}

func (jobLock *JobLock) Unlock() {
	if jobLock.isLocked {
		jobLock.cancelFunc()
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseId)
	}
}
