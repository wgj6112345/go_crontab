# go_crontab

基于 etcd 实现的分布式定时任务

## 使用方法

```
# 启动 master 节点

cd ./master/main
go build


# 启动 server 之前 请先开启 etcd 和 mongo 的服务
main.exe

# 管理界面在 http://localhost:10020



# 启动 worker 节点

cd ./worker/main
go build

main.exe





```
