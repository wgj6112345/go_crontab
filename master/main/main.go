package main

import (
	"flag"
	"fmt"

	"github.com/wgj6112345/go-crontab/go_crontab/master"
)

var (
	configFile string
)

func initArgs() {
	flag.StringVar(&configFile, "f", "./master.json", "specify config file")
	flag.Parse()
}

func main() {
	var (
		err error
	)
	// 初始化命令行参数
	initArgs()

	// 加载配置
	if err = master.InitConfig(configFile); err != nil {
		panic(err)
	}
	if err = master.InitLogMgr(); err != nil {
		panic(err)
	}
	if err = master.InitJobMgr(); err != nil {
		fmt.Println(err)
	}
	// worker 节点检查
	if err = master.InitWorkerMgr(); err != nil {
		fmt.Println(err)
	}

	// 启动 http 服务
	if err = master.InitApiServer(); err != nil {
		panic(err)
	}

	// 阻塞
	for {
	}
}
