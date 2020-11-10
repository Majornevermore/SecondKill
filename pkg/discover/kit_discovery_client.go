package discover

import (
	"SpeedKill/pkg/common"
	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"log"
	"strconv"
)

func NewConsulClientInstance(consulHost, consulPort string) *KitDiscoveryClient {
	port, _ := strconv.Atoi(consulPort)
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulHost + ":" + strconv.Itoa(port)
	apiClient, err := api.NewClient(consulConfig)
	if err != nil {
		return nil
	}
	client := consul.NewClient(apiClient)
	return &KitDiscoveryClient{
		Host: consulHost,
		Port: port,
		client: client,
		config: consulConfig,
	}
}

func (consulClient *KitDiscoveryClient) Register(instanceId, svcHost, healthCheckUrl, svcPort string, svcName string, weight int, meta map[string]string, tags []string, logger *log.Logger) bool {
	port, _ := strconv.Atoi(svcPort)
	serviceRegistration := &api.AgentServiceRegistration{
		ID: instanceId,
		Name: svcName,
		Address: svcHost,
		Port: port,
		Meta: meta,
		Tags: tags,
		Weights: &api.AgentWeights{
			Passing: weight,
		},
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "30",
			HTTP: "http://" + svcHost + ":" + strconv.Itoa(port) + healthCheckUrl,
			Interval: "15",
		},
	}
	err := consulClient.client.Register(serviceRegistration)
	if err != nil {
		if logger != nil {
			logger.Println("Register Service Error!")
		}
		return false
	}
	if logger != nil {
		logger.Println("Register Service Success!")
	}
	return true
}

func (consulClient *KitDiscoveryClient) DeRegister(instanceId string, logger *log.Logger) bool {
	serviceRegistration := &api.AgentServiceRegistration{
		ID: instanceId,
	}
	err := consulClient.client.Deregister(serviceRegistration)
	if err != nil {
		if logger != nil {
			logger.Println("Deregister Service Error!")
		}
		return false
	}
	if logger != nil {
		logger.Println("Deregister Service Success!")
	}
	return true
}

func (consulClient *KitDiscoveryClient) DiscoverServices(serviceName string, logger *log.Logger) []*common.ServiceInstance {
	if instanceList, ok := consulClient.instancesMap.Load(serviceName); ok {
		return instanceList.([]*common.ServiceInstance)
	}
	consulClient.mutex.Lock()
	// 再次检查是否监控
	if instanceList, ok := consulClient.instancesMap.Load(serviceName); ok {
		return instanceList.([]*common.ServiceInstance)
	} else {
		// 注册监控
		go func() {
			params := make(map[string]interface{})
			params["type"] = "service"
			params["service"] = serviceName
			plan, _ := watch.Parse(params)
			plan.Handler = func(u uint64, i interface{}) {
				if i == nil {
					return
				}
				// handler 方法中将Consul传递的数据转换为本地缓存的instanceMaps
				v, ok := i.([]*api.ServiceEntry)
				if !ok {
					return
				}
				// 没有服务在线
				if len(v) == 0 {
					consulClient.instancesMap.Store(serviceName, []*common.ServiceInstance{})
				}
				var healthServices []*common.ServiceInstance
				for _, service := range v {
					if service.Checks.AggregatedStatus() == api.HealthPassing {
						healthServices = append(healthServices, newServiceInstance(service.Service))
					}
				}
				consulClient.instancesMap.Store(serviceName, healthServices)
			}
			defer plan.Stop()
			plan.Run(consulClient.config.Address)
		}()

	}
	defer consulClient.mutex.Unlock()
	// 根据服务名获得请求实例
	entries, _, err := consulClient.client.Service(serviceName, "", false, nil)
	if err != nil {
		consulClient.instancesMap.Store(serviceName, []*common.ServiceInstance{})
		if logger != nil {
			logger.Println("Discover Service Error!")
		}
		return nil
	}
	instance := make([]*common.ServiceInstance, len(entries))
	for i:=0; i<len(instance); i++ {
		instance[i] = newServiceInstance(entries[i].Service)
	}
	consulClient.instancesMap.Store(serviceName, instance)
	return instance
}

func newServiceInstance(service *api.AgentService)  *common.ServiceInstance{
	rpcPort := service.Port - 1
	if service.Meta != nil {
		if rpcPortString, ok := service.Meta["rpcPort"]; ok {
			rpcPort, _ = strconv.Atoi(rpcPortString)
		}
	}
	return &common.ServiceInstance{
		Host:     service.Address,
		Port:     service.Port,
		GrpcPort: rpcPort,
		Weight:   service.Weights.Passing,
	}
}