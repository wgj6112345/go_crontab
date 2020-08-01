package worker

import (
	"time"

	"github.com/wgj6112345/go-crontab/go_crontab/common"
)

var (
	G_schedular Schedular
)

type Schedular struct {
	jobEventChan         chan *common.JobEvent
	jobExecuteResultChan chan *common.JobExecuteResult
	jobPlanTable         map[string]*common.SchedulePlan
	jobExecutingTable    map[string]*common.ExecuteInfo
}

func InitSchedular() {
	G_schedular = Schedular{
		jobEventChan:         make(chan *common.JobEvent, 1000),
		jobExecuteResultChan: make(chan *common.JobExecuteResult, 1000),
		jobPlanTable:         make(map[string]*common.SchedulePlan),
		jobExecutingTable:    make(map[string]*common.ExecuteInfo),
	}

	go G_schedular.Run()
}

// 调度器循环遍历
func (s *Schedular) Run() {
	var (
		interval time.Duration
		timer    *time.Timer
	)

	// 计算睡眠时间
	interval = s.ScanJobPlan()

	// 新建定时器
	timer = time.NewTimer(interval)

	for {
		select {
		case jobEvent := <-s.jobEventChan:
			s.HandleJobEvent(jobEvent)
		case <-timer.C:
		case jobExecuteResult := <-s.jobExecuteResultChan:
			// 返回执行结果，打印结果日志
			// TODO Kafka
			s.HandleJobExecuteResult(jobExecuteResult)
		}

		// jobPlanTable 如果有变化，要重新 Scan 一遍, 并且重置定时器
		interval = s.ScanJobPlan()
		timer.Reset(interval)
	}

}

// 扫描执行计划表，根据最近一次任务的执行时间，计算 Run() 协程中 的睡眠时间
func (s *Schedular) ScanJobPlan() (interval time.Duration) {
	var (
		jobPlan *common.SchedulePlan
		now     time.Time

		nearTime *time.Time // 最近一个快要过期的时间
	)

	// 任务列表为空，随机睡眠
	if len(s.jobPlanTable) == 0 {
		interval = 1 * time.Second
		return
	}

	// fmt.Println("jobPlanTable: ", s.jobPlanTable)
	for _, jobPlan = range s.jobPlanTable {
		now = time.Now()
		// 判断 jobPlan 有没有过期；
		// 如果过期，立即执行它；
		// 没有过期，计算最近一个即将过期的时间
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			s.TryExecuteJob(jobPlan)
			// 重新计算下一次的计划时间
			jobPlan.NextTime = jobPlan.Expr.Next(now)
		}

		// 没有过期
		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}
	}

	interval = (*nearTime).Sub(now)
	return
}

// 调度器遍历 任务列表，生成执行计划
func (s *Schedular) HandleJobEvent(jobEvent *common.JobEvent) {
	var (
		schedulePlan   *common.SchedulePlan
		err            error
		isExist        bool
		jobExecuteInfo *common.ExecuteInfo
		executing      bool
	)

	// 根据 每个 job 的 eventType 生成不同的执行计划
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:
		// 保存
		if schedulePlan, err = common.BuildSchedulePlan(jobEvent.Job); err != nil {
			return
		}
		s.jobPlanTable[jobEvent.Job.Name] = schedulePlan
	case common.JOB_EVENT_DELETE:
		// 删除   如果计划表有，删除；没有，不作处理
		if _, isExist = s.jobPlanTable[jobEvent.Job.Name]; isExist {
			delete(s.jobPlanTable, jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		// 强杀事件  删除 executingtable 中的 job
		if jobExecuteInfo, executing = s.jobExecutingTable[jobEvent.Job.Name]; executing {
			jobExecuteInfo.CancelFunc()
		}
	}
}

func (s *Schedular) HandleJobExecuteResult(jobExecuteResult *common.JobExecuteResult) {
	var (
		jobLog *common.JobLog
	)
	// 获取 执行结果后 删除 executeInfo 表中的 job
	delete(s.jobExecutingTable, jobExecuteResult.ExecuteInfo.Job.Name)
	// TODO: 将结果发往 日志处理 模块

	// 跳过 抢锁失败的 日志
	if jobExecuteResult.Err != common.ERR_JOB_LOCK_ALREADY_USED {
		jobLog = &common.JobLog{
			JobName:   jobExecuteResult.ExecuteInfo.Job.Name,
			Command:   jobExecuteResult.ExecuteInfo.Job.Command,
			Output:    jobExecuteResult.Result,
			PlanTime:  jobExecuteResult.ExecuteInfo.PlanTime.Format("2006-01-02 15:04:05.000"),
			RealTime:  jobExecuteResult.ExecuteInfo.RealTime.Format("2006-01-02 15:04:05.000"),
			StartTime: jobExecuteResult.StartTime,
			EndTime:   jobExecuteResult.EndTime,
		}

		if jobExecuteResult.Err != nil {
			jobLog.Err = jobExecuteResult.Err.Error()
		}
		G_logMgr.PushJobLog(jobLog)
	}

}

// 尝试执行任务
func (s *Schedular) TryExecuteJob(jobPlan *common.SchedulePlan) {
	var (
		isExecuting    bool
		jobExecuteInfo *common.ExecuteInfo
	)

	// fmt.Println("正在执行表: ", s.jobExecutingTable)
	if _, isExecuting = s.jobExecutingTable[jobPlan.Job.Name]; isExecuting {
		// 正在执行，跳过
		return
	}

	// 生成执行表 并保存
	jobExecuteInfo = common.BuildExecuteInfo(jobPlan)
	s.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo
	// 真正执行
	// fmt.Printf("正在执行任务：%v, planTime: %v, realTime: %v\n", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)

	// 开启协程执行
	G_execute.Execute(jobExecuteInfo)

}

// 调度到调度器
func (s *Schedular) PushJobEvent(jobEvent *common.JobEvent) {
	s.jobEventChan <- jobEvent
}

func (s *Schedular) PushJobExecuteResult(jobExecuteResult *common.JobExecuteResult) {
	s.jobExecuteResultChan <- jobExecuteResult
}
