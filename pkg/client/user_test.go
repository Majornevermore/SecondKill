package client

import (
	"context"
	"fmt"
	"SpeedKill/pb"
	_ "github.com/opentracing/opentracing-go"
	_ "github.com/openzipkin-contrib/zipkin-go-opentracing"
	_ "log"
	"testing"
)

func TestUserClientImpl_CheckUser(t *testing.T) {
	client, _ := NewUserClient("user", nil, nil)

	if response, err := client.CheckUser(context.Background(), nil, &pb.UserRequest{
		Username: "xuan",
		Password: "xuan",
	}); err == nil {
		fmt.Println(response.Result)
	} else {
		fmt.Println(err.Error())
	}
}
//
//func genTracerAct(tracer opentracing.Tracer) opentracing.Tracer {
//	if tracer != nil {
//		return tracer
//	}
//	zipkinUrl := "http://localhost:9411/api/v2/spans"
//	zipkinRecorder := "localhost:12344"
//	collector, err := zipkin.NewHTTPCollector(zipkinUrl)
//	if err != nil {
//		log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
//	}
//
//	recorder := zipkin.NewRecorder(collector, false, zipkinRecorder, "user-client")
//
//	res, err := zipkin.NewTracer(
//		recorder, zipkin.ClientServerSameSpan(true),
//	)
//	if err != nil {
//		log.Fatalf("zipkin.NewTracer err: %v", err)
//	}
//	return res
//
//}
