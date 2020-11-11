package mysql

import (
	"fmt"
	"github.com/gohouse/gorose/v2"
)

var engin *gorose.Engin
var err error

func InitMysql(hostMysql, portMysql, userMysql, pwdMysql, dbMysql string) {
	fmt.Printf(userMysql)
	fmt.Printf(dbMysql)
	DbConfig := gorose.Config{
		Driver: "mysql",
		Dsn:    userMysql + ":" + pwdMysql + "@tcp(" + hostMysql + ":" + portMysql + ")/" + dbMysql + "?charset=utf8&parseTime=true",
		Prefix: "", // Table prefix
		// (Connection pool) Max open connections, default value 0 means unlimit.
		SetMaxOpenConns: 300,
		// (Connection pool) Max idle connections, default value is 1.
		SetMaxIdleConns: 10,
	}
	engin, err = gorose.Open(&DbConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func DB() gorose.IOrm {
	return engin.NewOrm()
}
