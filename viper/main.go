package main

import (
	"fmt"
	"github.com/spf13/viper"
)


type YamlSetting struct{
	TimeStamp string
	Address string
	Postcode int64
	CompanyInfomation CompanyInfomation
}


type CompanyInfomation struct {
	Name string
	MarketCapitalization int64
	EmployeeNum int64
	Department []interface{}
	IsOpen bool
}

func parserYaml(v *viper.Viper)  {
	var yamlObj YamlSetting
	if err:= v.Unmarshal(&yamlObj); err != nil {
		fmt.Println(err)
	}
	fmt.Println(yamlObj)
}

func main()  {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath("./viper")
	v.AddConfigPath("$GOPATH/src/")
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		fmt.Println(err)
	}
	parserYaml(v)
}
