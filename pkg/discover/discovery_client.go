package discover

import (
	"SecondKill/pkg/common"
	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"log"
	"sync"
)

type KitDiscoveryClient struct {
	Host string //  Host
	Port int    //  Port
	// 连接 consul 的配置
	config *api.Config
	client consul.Client
	mutex  sync.Mutex
	// 服务实例缓存字段
	instancesMap sync.Map
}

type DiscoveryClient interface {
	Register(instanceId, svcHost, healthCheckUrl, svcPort string, svcName string, weight int, meta map[string]string, tags []string, logger *log.Logger) bool

	DeRegister(instaceID string, logger *log.Logger) bool

	DiscoverServices(serviceName string, logger *log.Logger) []*common.ServiceInstance
}
