package common

import "errors"

var (
	ERR_JOB_LOCK_ALREADY_USED = errors.New("锁已被占用")

	ERR_JOB_FORCE_STOP = errors.New("强制停止")

	ERR_JOB_EXECUTE_TIMEOUT = errors.New("运行超时")

	ERR_NO_LOCAL_IP_FOUND = errors.New("没有找到本地网卡")
)
