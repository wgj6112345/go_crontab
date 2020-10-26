package master

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/wgj6112345/go-crontab/go_crontab/common"
)

// http 接口
type ApiServer struct {
	httpServer *http.Server
}

// 创建单例

var (
	G_apiServer ApiServer
)

// 初始化 http 服务
func InitApiServer() (err error) {
	var (
		mux            *http.ServeMux
		listener       net.Listener
		server         *http.Server
		viewDir        http.Dir
		viewDirHandler http.Handler
	)

	mux = &http.ServeMux{}
	// 配置路由
	mux.HandleFunc("/job/save", handleJobSave)
	mux.HandleFunc("/job/delete", handleJobDelete)
	mux.HandleFunc("/job/list", handleJobList)
	mux.HandleFunc("/job/kill", handleJobKill)
	mux.HandleFunc("/job/log", handleJobLog)
	mux.HandleFunc("/worker/list", handleWorkerList)

	// 静态文件目录
	viewDir = http.Dir(G_config.ViewDir)
	viewDirHandler = http.FileServer(viewDir)
	mux.Handle("/", http.StripPrefix("/", viewDirHandler))

	if listener, err = net.Listen("tcp", fmt.Sprintf(":%d", G_config.ApiPort)); err != nil {
		fmt.Println(err)
		return
	}

	server = &http.Server{
		ReadTimeout:  time.Duration(G_config.ApiReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(G_config.ApiWriteTimeout) * time.Millisecond,
		Handler:      mux,
	}

	G_apiServer = ApiServer{
		httpServer: server,
	}

	go server.Serve(listener)

	fmt.Println("apiserver listening on port: ", G_config.ApiPort)
	return
}

func handleJobSave(w http.ResponseWriter, r *http.Request) {
	var (
		cronJob common.CronJob
		postJob string
		oldJob  common.CronJob
		err     error
		bytes   []byte
	)
	// 通过 etcd 保存    put job

	// 解析表单
	if err = r.ParseForm(); err != nil {
		fmt.Println(err)
	}

	postJob = r.PostForm.Get("job")

	fmt.Println("postJob: ", postJob)

	if err = json.Unmarshal([]byte(postJob), &cronJob); err != nil {
		goto ERR
	}

	if oldJob, err = G_jobMgr.SaveJob(&cronJob); err != nil {
		goto ERR
	}

	fmt.Println("oldJob: ", oldJob)
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
	return
}

func handleJobDelete(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		jobName string
		bytes   []byte
		oldJob  common.CronJob
	)
	// 解析表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	jobName = r.PostForm.Get("name")
	fmt.Println("jobName: ", jobName)

	// 删除 job
	if oldJob, err = G_jobMgr.DeleteJob(jobName); err != nil {
		goto ERR
	}

	fmt.Println("oldJob: ", oldJob)
	if bytes, err = common.BuildResponse(0, "success", oldJob); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
}

func handleJobList(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		bytes   []byte
		jobList []*common.CronJob
	)

	if jobList, err = G_jobMgr.ListJob(); err != nil {
		goto ERR
	}

	for _, job := range jobList {
		fmt.Println("job: ", job.Name)
	}

	if bytes, err = common.BuildResponse(0, "success", jobList); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
	return
}

func handleJobKill(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		jobName string
		bytes   []byte
	)

	// 解析表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	jobName = r.PostForm.Get("name")

	if err = G_jobMgr.KillJob(jobName); err != nil {
		goto ERR
	}
	if bytes, err = common.BuildResponse(0, "success", nil); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
	return
}

func handleJobLog(w http.ResponseWriter, r *http.Request) {
	var (
		err        error
		jobName    string
		jobLogList []*common.JobLog
		bytes      []byte
	)
	// 解析表单
	if err = r.ParseForm(); err != nil {
		goto ERR
	}

	// 获取参数
	jobName = r.PostForm.Get("name")

	if jobLogList, err = G_logMgr.ListLogs(jobName, G_config.LogSkip, G_config.LogLimit); err != nil {
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "success", jobLogList); err == nil {
		w.Write(bytes)
	}
	return

ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
	return
}

func handleWorkerList(w http.ResponseWriter, r *http.Request) {
	var (
		workerList []string
		err        error
		bytes      []byte
	)

	if workerList, err = G_workerMgr.ListWorkers(); err != nil {
		goto ERR
	}

	if bytes, err = common.BuildResponse(0, "success", workerList); err == nil {
		w.Write(bytes)
	}
	return
ERR:
	if bytes, err = common.BuildResponse(-1, err.Error(), nil); err == nil {
		w.Write(bytes)
	}
	return
}
