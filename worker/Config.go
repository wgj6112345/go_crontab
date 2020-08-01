package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var (
	G_config *Config
)

type Config struct {
	EtcdPoints          []string `json:"etcd_points"`
	EtcdDialTimeout     int      `json:"etcd_dial_timeout"`
	MongoHost           string   `json:"mongodb_host"`
	MongoConnectTimeout int      `json:"mongodb_connect_timeout"`
	MongoDataBase       string   `json:"mongodb_database"`
	MongoCollection     string   `json:"mongodb_collection"`
	LogBatchSize        int      `json:"log_batch_size"`
	LogCommitTimeout    int      `json:"log_commit_timeout"`
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

	fmt.Println("worker 配置初始化成功, config: ", G_config)
	return
}
