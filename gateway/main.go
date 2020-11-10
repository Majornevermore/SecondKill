package main

import (
	"SpeedKill/gateway/router"
	"SpeedKill/pkg/bootstrap"
	register "SpeedKill/pkg/discover"
	"flag"
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/go-kit/kit/log"
	"github.com/openzipkin/zipkin-go"
	zipkinhttpsvr "github.com/openzipkin/zipkin-go/middleware/http"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main()  {
	// 创建环境变量
	var (
		zipkinURL = flag.String("zipkin.url", "http://localhost:9411/api/v2/spans", "Zipkin server url")
	)
	flag.Parse()

	//创建日志组件
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	var zipkinTracer *zipkin.Tracer
	{
		var (
			err           error
			useNoopTracer = *zipkinURL == ""
			reporter      = zipkinhttp.NewReporter(*zipkinURL)
		)
		defer reporter.Close()
		zEP, _ := zipkin.NewEndpoint(bootstrap.HttpConfig.Host, bootstrap.HttpConfig.Port)
		zipkinTracer, err = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(zEP), zipkin.WithNoopTracer(useNoopTracer),
		)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		if !useNoopTracer {
			logger.Log("tracer", "Zipkin", "type", "Native", "URL", *zipkinURL)
		}
	}
	register.Register()
	tags := map[string]string{
		"component": "gateway_server",
	}
	hystrixRouter := router.Router(zipkinTracer, "Circuit Breaker:Service unavailable", logger)
	handle := zipkinhttpsvr.NewServerMiddleware(zipkinTracer, zipkinhttpsvr.SpanName(bootstrap.DiscoverConfig.ServiceName),
		zipkinhttpsvr.TagResponseSize(true),
		zipkinhttpsvr.ServerTags(tags))(hystrixRouter)
	errc := make(chan error)
	//启用hystrix实时监控，监听端口为9010
	hystrixStreamHandler := hystrix.NewStreamHandler()
	hystrixStreamHandler.Start()
	go func() {
		errc <- http.ListenAndServe(net.JoinHostPort("", "9010"), hystrixStreamHandler)
	}()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()
	go func() {
		logger.Log("transport", "http", "add", "9090")
		register.Register()
		errc <- http.ListenAndServe("9090", handle)
	}()
	err := <- errc
	register.DeRegister()
	logger.Log("exit", err)
}
