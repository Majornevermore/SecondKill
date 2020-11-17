package setup

import (
	conf "SecondKill/pkg/config"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"time"
)

// chushihu

func InitZk() {
	var host = []string{"39.98.179.73:2181"}
	conn, _, err := zk.Connect(host, time.Second*5)
	if err != nil {
		fmt.Println(err)
		return
	}
	conf.Zk.ZkConn = conn
	conf.Zk.SecProductKey = "/product"
}
