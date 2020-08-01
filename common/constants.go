package common

const (
	CRON_JOB_DIR      = "/cron/job/"
	CRON_KILL_JOB_DIR = "/cron/kill/"
	CRON_JOB_LOCK_DIR = "/cron/lock/"
	CRON_WORKER_DIR   = "/cron/worker/"
)

const (
	JOB_EVENT_SAVE   = 1
	JOB_EVENT_DELETE = 2
	JOB_EVENT_KILL   = 3
)
