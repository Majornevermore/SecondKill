package discover

import (
	"SecondKill/pkg/bootstrap"
	"SecondKill/pkg/common"
	"SecondKill/pkg/loadbalance"
	"errors"
	uuid "github.com/satori/go.uuid"
	"log"
	"os"
)

var (
	ConsulService DiscoveryClient
	Logger        *log.Logger
	BalanceS      loadbalance.Balance
)

var NoInstanceExistedErr = errors.New("no available client")

func init() {
	ConsulService = NewConsulClientInstance(bootstrap.DiscoverConfig.Host, bootstrap.DiscoverConfig.Port)
	Logger = log.New(os.Stderr, "", log.LstdFlags)
	BalanceS = new(loadbalance.RandomBalance)
}

func Discover(service string) (*common.ServiceInstance, error) {
	if ConsulService == nil {
		return nil, NoInstanceExistedErr
	}
	instances := ConsulService.DiscoverServices(service, Logger)
	if len(instances) < 1 {
		return nil, NoInstanceExistedErr
	}
	return BalanceS.SelectBalance(instances)
}

func Register() {
	instance := bootstrap.DiscoverConfig.InstanceId
	if instance == "" {
		instance = bootstrap.DiscoverConfig.ServiceName + uuid.NewV4().String()
	}
	if !ConsulService.Register(instance, bootstrap.HttpConfig.Host, "/health",
		bootstrap.HttpConfig.Port, bootstrap.DiscoverConfig.ServiceName, bootstrap.DiscoverConfig.Weight,
		map[string]string{
			"rpcPort": bootstrap.RpcConfig.Port,
		}, nil, Logger) {
		Logger.Printf("register service %s failed.", bootstrap.DiscoverConfig.ServiceName)
		// 注册失败，服务启动失败
		panic(0)
	}
	Logger.Printf(bootstrap.DiscoverConfig.ServiceName+"-service for service %s success.", bootstrap.DiscoverConfig.ServiceName)
}

func DeRegister() {
	if ConsulService == nil {
		return
	}
	instance := bootstrap.DiscoverConfig.InstanceId
	if instance == "" {
		instance = bootstrap.DiscoverConfig.ServiceName + uuid.NewV4().String()
	}
	if !ConsulService.DeRegister(instance, Logger) {
		Logger.Printf("deregister for service %s failed.", bootstrap.DiscoverConfig.ServiceName)
		panic(0)
	}
}
