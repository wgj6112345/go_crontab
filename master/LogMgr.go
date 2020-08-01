package master

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
	client     *mongo.Client
	collection *mongo.Collection
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
		client:     client,
		collection: collection,
	}

	return
}

// 获取日志
func (logMgr *LogMgr) ListLogs(jobName string, skipNum int64, limitNum int64) (jobLogList []*common.JobLog, err error) {
	var (
		cond   *common.JobLogFilterByName
		cursor *mongo.Cursor
		jobLog *common.JobLog
		sort   *common.JobLogSortByTime
	)
	// 定义过滤条件
	cond = &common.JobLogFilterByName{JobName: jobName}
	//排序规则
	sort = &common.JobLogSortByTime{PlanTime: -1}

	if cursor, err = logMgr.collection.Find(context.TODO(), cond, &options.FindOptions{Sort: sort}, &options.FindOptions{Skip: &skipNum}, &options.FindOptions{Limit: &limitNum}); err != nil {
		fmt.Println(err)
		return
	}
	defer cursor.Close(context.TODO())
	// 遍历结果集
	for cursor.Next(context.TODO()) {
		jobLog = &common.JobLog{}
		if err = cursor.Decode(jobLog); err != nil {
			fmt.Println(err)
			return
		}

		jobLogList = append(jobLogList, jobLog)
	}

	return
}
