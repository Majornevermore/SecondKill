package router

import (
	"SpeedKill/gateway/config"
	"SpeedKill/pb"
	"SpeedKill/pkg/client"
	"SpeedKill/pkg/discover"
	"SpeedKill/pkg/loadbalance"
	"context"
	"errors"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/log"
	"github.com/openzipkin/zipkin-go"
	zipkinhttpsvr "github.com/openzipkin/zipkin-go/middleware/http"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
)

type HystrixRouter struct {
	svcMap *sync.Map // 服务实例，存储已通过hystrix监控
	log log.Logger // 日志工具
	fallbackMsg string // 回调消息
	tracer *zipkin.Tracer
	loadbalance loadbalance.Balance
}

func Router(zipTracer *zipkin.Tracer, fbMsg string, logger log.Logger) http.Handler {
	return HystrixRouter{
		svcMap: &sync.Map{},
		log: logger,
		fallbackMsg: fbMsg,
		tracer: zipTracer,
		loadbalance: &loadbalance.RandomBalance{},
	}
}

func preFilter(r *http.Request) bool {
	reqPath := r.URL.Path
	if reqPath == "" {
		return false
	}
	res := config.Match(reqPath)
	if res {
		return true
	}
	authToken := r.Header.Get("Authorization")
	if authToken == "" {
		return false
	}
	oathClient, _ := client.NewOAuthClient("oauth", nil, nil)
	resp, remoteErr := oathClient.CheckToken(context.Background(), nil, &pb.CheckTokenRequest{
		Token: authToken,
	})
	if remoteErr != nil || resp == nil {
		return false
	}
	return true
}

func (router HystrixRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path
	router.log.Log("reqPath: ", reqPath)
	// 健康检查直接返回
	if reqPath == "/health" {
		w.WriteHeader(200)
		return
	}
	var err error
	if reqPath == "" || !preFilter(r) {
		err = errors.New("illegal request!")
		w.WriteHeader(403)
		w.Write([]byte(err.Error()))
		return
	}
	//按照分隔符'/'对路径进行分解，获取服务名称serviceName
	pathArray := strings.Split(reqPath, "/")
	serviceName := pathArray[1]

	if _, ok := router.svcMap.Load(serviceName); !ok {
		hystrix.ConfigureCommand(serviceName, hystrix.CommandConfig{
			Timeout: 1000,
		})
		router.svcMap.Store(serviceName, serviceName)
	}

	// 执行命令
	err = hystrix.Do(serviceName, func() error {
		serviceInstance, err := discover.Discover(serviceName)
		if err != nil {
			return err
		}
		director := func(request *http.Request) {
			despath := strings.Join(pathArray[2:], "/")
			router.log.Log("serive id", serviceInstance.Host, serviceInstance.Port)
			request.URL.Scheme = "http"
			request.URL.Host = fmt.Sprintf("%s:%d", serviceInstance.Host, serviceInstance.Port)
			request.URL.Path = "/" + despath
		}
		var proxyError error = nil
		roundTip, _ := zipkinhttpsvr.NewTransport(router.tracer, zipkinhttpsvr.TransportTrace(true))
		errorHandle := func(ew http.ResponseWriter, er *http.Request, err error) {
			proxyError = err
		}
		proxy := &httputil.ReverseProxy{
			Director: director,
			Transport : roundTip,
			ErrorHandler: errorHandle,
		}
		proxy.ServeHTTP(w, r)
		return proxyError
	}, func(err error) error {
		//run执行失败，返回fallback信息
		router.log.Log("fallback error description", err.Error())

		return errors.New(router.fallbackMsg)
	})
	// Do方法执行失败，响应错误信息
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
}



