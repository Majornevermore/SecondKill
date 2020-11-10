package client

import (
	_ "SpeedKill/pkg/bootstrap"
	"SpeedKill/pkg/discover"
	"SpeedKill/pkg/loadbalance"
	_ "SpeedKill/pkg/config"
	"context"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	_  "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"google.golang.org/grpc"
	"log"
	"strconv"
	"errors"
	"time"
)

var defaultLoadBalance loadbalance.Balance = &loadbalance.RandomBalance{}
var (
	ErrRPCService = errors.New("no rpc service")
)
type ClientManager interface {
	DecoratorInvoke(path string, hystrixName string, tracer opentracing.Tracer,
		ctx context.Context, inputVal interface{}, outVal interface{}) (err error)
}

type DefaultClientManager struct{
	serviceName string
	logger *log.Logger
	discoverClient discover.DiscoveryClient
	loadbalance loadbalance.Balance
	after []InvokerAfterFunc
	before []InvokerBeforeFunc
}

type InvokerAfterFunc func() (err error)

type InvokerBeforeFunc func() (err error)

func (manager *DefaultClientManager) DecoratorInvoke(path string, hystrixName string, tracer opentracing.Tracer, ctx context.Context, inputVal interface{}, outVal interface{}) (err error) {
	for _, fn := range manager.before {
		if err = fn(); err != nil {
			return err
		}
	}
	if err = hystrix.Do(hystrixName, func() error {
		instances := manager.discoverClient.DiscoverServices(manager.serviceName, manager.logger)
		if instances, err := manager.loadbalance.SelectBalance(instances); err == nil {
			if instances.GrpcPort > 0 {
				if conn, err := grpc.Dial(instances.Host+":"+strconv.Itoa(instances.Port), grpc.WithInsecure(),
					grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(genTracer(tracer),
						otgrpc.LogPayloads())), grpc.WithTimeout(1*time.Second)); err == nil {
					if err = conn.Invoke(ctx, path, inputVal, outVal); err != nil {
						return err
					}
				}
			} else {
				return ErrRPCService
			}
		} else {
			return err
		}
		return nil
	}, func(e error) error {
		return e
	}); err != nil {
		return err
	} else {
		for _, fn := range manager.after {
			if err = fn(); err != nil {
				return err
			}
		}
		return nil
	}
}

func genTracer(tracer opentracing.Tracer) opentracing.Tracer {
	if tracer != nil {
		return tracer
	}
	return nil
	//zipkinUrl := "http://" + conf.TraceConfig.Host + ":" + conf.TraceConfig.Port + conf.TraceConfig.Url
	//zipkinRecorder := bootstrap.HttpConfig.Host + ":" + bootstrap.HttpConfig.Port
	//collector, err := zipkin.(zipkinUrl)
	//if err != nil {
	//	log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
	//}
	//
	//recorder := zipkin.NewRecorder(collector, false, zipkinRecorder, bootstrap.DiscoverConfig.ServiceName)
	//
	//res, err := zipkin.NewTracer(
	//	recorder, zipkin.ClientServerSameSpan(true),
	//)
	//if err != nil {
	//	log.Fatalf("zipkin.NewTracer err: %v", err)
	//}
	//return res

}
