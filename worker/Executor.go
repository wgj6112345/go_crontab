package worker

import (
	"fmt"
	"math/rand"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/wgj6112345/go-crontab/go_crontab/common"
)

var (
	G_execute *Executor
)

type Executor struct{}

func InitExecutor() {
	G_execute = &Executor{}
}

func (e *Executor) Execute(jobExecuteInfo *common.ExecuteInfo) {
	// 开启协程执行
	go func() {
		var (
			jobLock        *JobLock
			cmd            *exec.Cmd
			command        string
			splitStr       []string
			bash           string
			args           string
			output         []byte
			err            error
			jobEventResult *common.JobExecuteResult
			jobTimer       *time.Timer
			doneChan       = make(chan *common.JobExecuteResult)
		)

		// 这样写会报错
		// jobEventResult.ExecuteInfo = jobExecuteInfo
		jobEventResult = &common.JobExecuteResult{
			ExecuteInfo: jobExecuteInfo,
		}
		jobEventResult.StartTime = time.Now().Format("2006-01-02 15:04:05.000")
		// 获取分布式锁
		// 	  为避免锁占用倾斜问题，随机睡眠 1000 ms
		rand.Seed(time.Now().UnixNano())
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)
		jobLock = G_jobMgr.CreateJobLock(jobExecuteInfo.Job.Name)

		err = jobLock.TryLock()
		defer jobLock.Unlock()

		if err != nil {
			// 抢锁失败
			fmt.Println("抢锁失败，跳过执行")
			jobEventResult.Err = err
			jobEventResult.EndTime = time.Now().Format("2006-01-02 15:04:05.000")
		} else {
			// 抢锁成功 运行 command
			fmt.Println("抢锁成功！")

			// 真正执行
			jobEventResult.StartTime = time.Now().Format("2006-01-02 15:04:05.000")

			go func(jobExecuteInfo *common.ExecuteInfo) {
				splitStr = strings.SplitN(jobExecuteInfo.Job.Command, " ", 2)
				bash = splitStr[0]
				args = splitStr[1]

				// 如果是 windows
				if runtime.GOOS == "windows" {
					command = fmt.Sprintf("C:/cygwin64/bin/%s.exe", bash)
					cmd = exec.CommandContext(jobExecuteInfo.CancelCtx, command, args)
				}
				// 如果是 linux
				if runtime.GOOS == "linux" {
					cmd = exec.CommandContext(jobExecuteInfo.CancelCtx, bash, args)
				}

				// 模拟运行超时 5s
				time.Sleep(time.Second * 5)
				if output, err = cmd.CombinedOutput(); err != nil {
					err = common.ERR_JOB_FORCE_STOP
				}

				jobEventResult.EndTime = time.Now().Format("2006-01-02 15:04:05.000")
				jobEventResult.Result = string(output)
				jobEventResult.Err = err

				fmt.Printf("%v \n执行结果: %v \nERR: %v \nstarTime: %v, \nendTime: %v\n", jobEventResult.ExecuteInfo.Job.Name, string(output), err, jobEventResult.StartTime, jobEventResult.EndTime)

				doneChan <- jobEventResult
			}(jobExecuteInfo)

			jobTimer = time.NewTimer(time.Duration(jobExecuteInfo.Job.JobTimeout) * time.Second)

			select {
			case jobEventResult = <-doneChan:
				fmt.Println("正常执行完成")
				G_schedular.PushJobExecuteResult(jobEventResult)
				jobTimer.Stop()
			case <-jobTimer.C:
				fmt.Println("进入定时器")
				jobExecuteInfo.CancelFunc()
				jobTimer.Stop()

				jobEventResult.Err = common.ERR_JOB_EXECUTE_TIMEOUT
				jobEventResult.Result = ""
				jobEventResult.EndTime = time.Now().Format("2006-01-02 15:04:05.000")

				G_schedular.PushJobExecuteResult(jobEventResult)
			}

		}
	}()

}
