package worker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/wgj6112345/go-crontab/go_crontab/common"

	"github.com/coreos/etcd/clientv3"
)

var (
	G_registry Registry
)

type Registry struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease

	localIp string
}

func InitRegistry() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		localIp net.IP
	)

	config = clientv3.Config{
		Endpoints:   G_config.EtcdPoints,
		DialTimeout: time.Duration(G_config.EtcdDialTimeout) * time.Millisecond,
	}

	if client, err = clientv3.New(config); err != nil {
		fmt.Println(err)
		return
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)

	if localIp, err = externalIP(); err != nil {
		fmt.Println(err)
		return
	}

	G_registry = Registry{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIp: localIp.String(),
	}

	go G_registry.Register()
	return

}

// 注册本机地址到 etcd
func (r *Registry) Register() {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		err            error
		leaseId        clientv3.LeaseID
		ctx            context.Context
		cancelFunc     context.CancelFunc
		leaseKeepChan  <-chan *clientv3.LeaseKeepAliveResponse
		leaseKeep      *clientv3.LeaseKeepAliveResponse
		workerKey      string
	)

	// 中途注册过程中失败的话，可以重试
	for {
		// 获取 租约
		if leaseGrantResp, err = r.lease.Grant(context.TODO(), 10); err != nil {
			goto RETRY
		}

		leaseId = leaseGrantResp.ID

		ctx, cancelFunc = context.WithCancel(context.TODO())
		if leaseKeepChan, err = r.lease.KeepAlive(ctx, leaseId); err != nil {
			goto RETRY
		}

		// 把 localIp 注册到 etcd
		workerKey = fmt.Sprintf("%s%s", common.CRON_WORKER_DIR, r.localIp)

		if _, err = r.kv.Put(context.TODO(), workerKey, "", clientv3.WithLease(leaseId)); err != nil {
			goto RETRY
		}

		// 处理续租应答
		for {
			select {
			case leaseKeep = <-leaseKeepChan:
				if leaseKeep == nil {
					goto RETRY
				}
			}
		}
		// 中途注册过程中失败的话，可以重试
	RETRY:
		time.Sleep(time.Second)
		if cancelFunc != nil {
			cancelFunc()
		}
	}
}

func getLocalIp() (ipv4 string, err error) {
	var (
		ipList  []net.Addr
		ip      net.Addr
		ipNet   *net.IPNet
		isIPNet bool
	)
	if ipList, err = net.InterfaceAddrs(); err != nil {
		fmt.Println(err)
		return
	}

	for _, ip = range ipList {
		// ip 是一个 interface
		if ipNet, isIPNet = ip.(*net.IPNet); isIPNet && !ipNet.IP.IsLoopback() {
			// 跳过 IPv6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()
				fmt.Println("ipv4: ", ipv4)
				return
			}
		}
	}

	// 没有找到 ipv4 的地址，返回错误
	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}

func externalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}
