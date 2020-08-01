package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/wgj6112345/go-crontab/go_crontab/common"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	G_logMgr LogMgr
)

type LogMgr struct {
	client         *mongo.Client
	collection     *mongo.Collection
	logChan        chan *common.JobLog
	autoCommitChan chan *common.JobLogBatch
}

func InitLogMgr() (err error) {
	var (
		mongodbUri string
		opts       *options.ClientOptions
		client     *mongo.Client
		database   *mongo.Database
		collection *mongo.Collection
	)

	// 连接mongodb
	mongodbUri = fmt.Sprintf("mongodb://%s", G_config.MongoHost)
	opts = options.Client().ApplyURI(mongodbUri)
	opts.SetConnectTimeout(time.Duration(G_config.MongoConnectTimeout) * time.Millisecond)

	if client, err = mongo.Connect(context.TODO(), opts); err != nil {
		fmt.Println(err)
		return
	}

	if err = client.Ping(context.TODO(), nil); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("connect mongodb host: %v success\n", mongodbUri)

	// 选择数据库
	database = client.Database(G_config.MongoDataBase)
	// 选择数据表
	collection = database.Collection(G_config.MongoCollection)

	G_logMgr = LogMgr{
		client:         client,
		collection:     collection,
		logChan:        make(chan *common.JobLog, 1000),
		autoCommitChan: make(chan *common.JobLogBatch, 1000),
	}

	G_logMgr.Run()
	return
}

func (logMgr *LogMgr) Run() {
	go func() {

		var (
			jobLog         *common.JobLog
			jobLogBatch    *common.JobLogBatch
			timeoutBatch   *common.JobLogBatch
			autoCommitTime *time.Timer
		)
		for {
			select {
			case jobLog = <-logMgr.logChan:
				// 保存 jobLog 到 mongo
				// 批量插入
				if jobLogBatch == nil {
					jobLogBatch = &common.JobLogBatch{}

					// 超时自动提交日志
					//     开启协程的时候，如果用到函数外的参数，不能直接使用，应该通过函数参数进行传入
					//     如果不传入的话，你在使用该参数前，它的值很可能已经被其他协程修改掉了
					autoCommitTime = time.AfterFunc(time.Duration(G_config.LogCommitTimeout)*time.Millisecond, func(jobLogBatch *common.JobLogBatch) func() {
						return func() {
							logMgr.autoCommitChan <- jobLogBatch
						}
					}(jobLogBatch))
				}

				jobLogBatch.Logs = append(jobLogBatch.Logs, jobLog)
				// 每 100 条 写入一次
				if len(jobLogBatch.Logs) > G_config.LogBatchSize {
					logMgr.saveLogs(jobLogBatch)
					jobLogBatch = nil
					// 关闭定时器
					autoCommitTime.Stop()
				}
			case timeoutBatch = <-logMgr.autoCommitChan:
				// 满 100 条提交后，如果定时器这个时候也提交，那么会提交失败
				if timeoutBatch != jobLogBatch {
					continue
				}
				logMgr.saveLogs(timeoutBatch)
				jobLogBatch = nil
			}
		}
	}()

}

func (logMgr *LogMgr) saveLogs(logBatch *common.JobLogBatch) {
	var err error
	if _, err = logMgr.collection.InsertMany(context.TODO(), logBatch.Logs); err != nil {
		fmt.Println("save logbatch failed, err: ", err)
	}
	fmt.Println("save logs batch success")
}

func (logMgr *LogMgr) PushJobLog(jobLog *common.JobLog) {
	// mongodb 写入过慢的话，其他日志无法写入，作丢弃处理
	select {
	case logMgr.logChan <- jobLog:
	default:
	}
}
