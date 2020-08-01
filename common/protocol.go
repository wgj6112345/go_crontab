package common

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorhill/cronexpr"
)

// 任务名  shell命令  cron表达式
type CronJob struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	CronExpr   string `json:"cron_expr"`
	JobTimeout int    `json:"job_timeout"`
}

type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

func BuildResponse(errNo int, msg string, data interface{}) (bytes []byte, err error) {
	var (
		response Response
	)

	response.Errno = errNo
	response.Msg = msg
	response.Data = data

	bytes, err = json.Marshal(response)
	return
}

type JobEvent struct {
	EventType int // 0 get  1 put -1 delete
	Job       *CronJob
}

func BuildJobEvent(eventType int, job *CronJob) *JobEvent {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

type SchedulePlan struct {
	Job      *CronJob
	Expr     *cronexpr.Expression
	NextTime time.Time
}

// worker 的执行器 根据 这个 schedulePlan 表进行执行
func BuildSchedulePlan(job *CronJob) (schedulePlan *SchedulePlan, err error) {
	var (
		cronExpr *cronexpr.Expression
		now      time.Time = time.Now()
		nextTime time.Time
	)

	if cronExpr, err = cronexpr.Parse(job.CronExpr); err != nil {
		return
	}
	nextTime = cronExpr.Next(now)
	schedulePlan = &SchedulePlan{
		Job:      job,
		Expr:     cronExpr,
		NextTime: nextTime,
	}
	return
}

type ExecuteInfo struct {
	Job      *CronJob
	PlanTime time.Time // 计划执行时间
	RealTime time.Time // 实际执行时间

	CancelCtx  context.Context
	CancelFunc context.CancelFunc
}

func BuildExecuteInfo(jobPlan *SchedulePlan) *ExecuteInfo {
	var (
		ctx        context.Context
		cancelFunc context.CancelFunc
	)

	ctx, cancelFunc = context.WithCancel(context.TODO())
	return &ExecuteInfo{
		Job:        jobPlan.Job,
		PlanTime:   jobPlan.NextTime,
		RealTime:   time.Now(),
		CancelCtx:  ctx,
		CancelFunc: cancelFunc,
	}
}

type JobExecuteResult struct {
	ExecuteInfo *ExecuteInfo
	Result      string
	Err         error
	StartTime   string
	EndTime     string
}

// 反序列化
func Unmarshal(content []byte) (job *CronJob, err error) {
	var (
		cronJob CronJob
	)

	if err = json.Unmarshal(content, &cronJob); err != nil {
		fmt.Println("err: ", err)
	}

	job = &cronJob
	return
}

type JobLog struct {
	JobName   string `json:"jobname" bson:"jobname"`
	Command   string `json:"command" bson:"command"`
	Output    string `json:"output" bson:"output"`
	Err       string `json:"err" bson:"err"`
	PlanTime  string `json:"plan_time" bson:"plan_time"`
	RealTime  string `json:"real_time" bson:"real_time"`
	StartTime string `json:"start_time" bson:"start_time"`
	EndTime   string `json:"end_time" bson:"end_time"`
}

// 日志批次
type JobLogBatch struct {
	Logs []interface{} //JobLog
}

// 定义日志过滤条件
type JobLogFilterByName struct {
	JobName string `bson:"jobname"`
}

type JobLogSortByTime struct {
	PlanTime int `bson:"plan_time"`
}
