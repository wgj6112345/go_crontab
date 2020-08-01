package master

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var (
	G_config *Config
)

type Config struct {
	ApiPort             int      `json:"api_port"`
	ApiReadTimeout      int      `json:"api_read_timeout"`
	ApiWriteTimeout     int      `json:"api_write_timeout"`
	EtcdPoints          []string `json:"etcd_points"`
	EtcdDialTimeout     int      `json:"etcd_dial_timeout"`
	ViewDir             string   `json:"view_dir"`
	MongoHost           string   `json:"mongodb_host"`
	MongoConnectTimeout int      `json:"mongodb_connect_timeout"`
	MongoDataBase       string   `json:"mongodb_database"`
	MongoCollection     string   `json:"mongodb_collection"`
	LogSkip             int64    `json:"job_log_skip"`
	LogLimit            int64    `json:"job_log_limit"`
}

func InitConfig(filename string) (err error) {
	var (
		content []byte
		conf    Config
	)

	if content, err = ioutil.ReadFile(filename); err != nil {
		fmt.Println(err)
		return
	}

	if err = json.Unmarshal(content, &conf); err != nil {
		fmt.Println(err)
		return
	}

	G_config = &conf
	return
}
