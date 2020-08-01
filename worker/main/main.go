package main

import (
	"flag"
	"runtime"

	"github.com/wgj6112345/go-crontab/go_crontab/worker"
)

var (
	configFile string
)

func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func initArgs() {
	flag.StringVar(&configFile, "f", "./worker.json", "specify config file")
	flag.Parse()
}

func main() {
	var (
		err error
	)
	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	// 加载配置
	if err = worker.InitConfig(configFile); err != nil {
		panic(err)
	}

	// 启动日志处理
	if err = worker.InitLogMgr(); err != nil {
		panic(err)
	}

	// 启动服务注册
	if err = worker.InitRegistry(); err != nil {
		panic(err)
	}

	// 启动执行器
	worker.InitExecutor()

	// 启动 schedular
	worker.InitSchedular()

	// 启动 jobMrg
	if err = worker.InitJobMgr(); err != nil {
		panic(err)
	}

	//
	for {
	}
}
