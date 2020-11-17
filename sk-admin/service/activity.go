package service

import (
	conf "SecondKill/pkg/config"
	"SecondKill/sk-admin/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gohouse/gorose/v2"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/unknwon/com"
	"log"
	"time"
)

type ActivityService interface {
	GetActivityList() ([]gorose.Data, error)
	CreateActivity(activity *model.Activity) error
}

type ActivityServiceImpl struct{}

func (p ActivityServiceImpl) GetActivityList() ([]gorose.Data, error) {
	activityEntity := model.NewActivityModel()
	activityList, err := activityEntity.GetActivityList()
	if err != nil {
		log.Printf("ActivityEntity.GetActivityList, err : %v", err)
		return nil, err
	}

	for _, v := range activityList {
		startTime, _ := com.StrTo(fmt.Sprint(v["start_time"])).Int64()
		v["start_time_str"] = time.Unix(startTime, 0).Format("2006-01-02 15:04:05")

		endTime, _ := com.StrTo(fmt.Sprint(v["end_time"])).Int64()
		v["end_time_str"] = time.Unix(endTime, 0).Format("2006-01-02 15:04:05")

		nowTime := time.Now().Unix()
		if nowTime > endTime {
			v["status_str"] = "已结束"
			continue
		}

		status, _ := com.StrTo(fmt.Sprint(v["status"])).Int()
		if status == model.ActivityStatusNormal {
			v["status_str"] = "正常"
		} else if status == model.ActivityStatusDisable {
			v["status_str"] = "已禁用"
		}
	}

	log.Printf("get activity success, activity list is [%v]", activityList)
	return activityList, nil

}

func (p ActivityServiceImpl) CreateActivity(activity *model.Activity) error {
	log.Printf("CreateActivity")
	activityEntity := model.NewActivityModel()
	err := activityEntity.CreateActivity(activity)
	if err != nil {
		log.Printf("err %v", err)
	}
	log.Printf("syncTozk")

	if err != nil {
		log.Printf("%v", err)
		return err
	}
	return nil
}

func (p ActivityServiceImpl) loadProductFromZk(key string) ([]*model.SecProductInfoConf, error) {
	_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	v, s, err := conf.Zk.ZkConn.Get(key)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}
	log.Printf("get from zk success, rsp : %v", s)
	var secProductInfo []*model.SecProductInfoConf
	err1 := json.Unmarshal(v, &secProductInfo)
	if err1 != nil {
		log.Printf("load product from zk err %v", err1)
		return nil, err1
	}
	return secProductInfo, nil
}

func (p ActivityServiceImpl) syncToZk(activity *model.Activity) error {
	zkPath := conf.Zk.SecProductKey
	secProductInfoList, err := p.loadProductFromZk(zkPath)
	if err != nil {
		secProductInfoList = []*model.SecProductInfoConf{}
	}
	var secProductInfo = &model.SecProductInfoConf{}
	secProductInfo.EndTime = activity.EndTime
	secProductInfo.OnePersonBuyLimit = activity.BuyLimit
	secProductInfo.ProductId = activity.ProductId
	secProductInfo.SoldMaxLimit = activity.Speed
	secProductInfo.StartTime = activity.StartTime
	secProductInfo.Status = activity.Status
	secProductInfo.Total = activity.Total
	secProductInfo.BuyRate = activity.BuyRate
	secProductInfoList = append(secProductInfoList, secProductInfo)
	data, err := json.Marshal(secProductInfoList)
	if err != nil {
		log.Printf("json marshal failed, err : %v", err)
		return err
	}
	conn := conf.Zk.ZkConn

	var byteData = []byte(string(data))
	var flags int32 = 0
	var acls = zk.WorldACL(zk.PermAll)
	exisits, _, _ := conn.Exists(zkPath)
	if exisits {
		_, err_set := conn.Set(zkPath, byteData, flags)
		if err_set != nil {
			fmt.Println(err_set)
		}
	} else {
		_, err_create := conn.Create(zkPath, byteData, flags, acls)
		if err_create != nil {
			fmt.Println(err_create)
		}
	}
	log.Printf("put to zk success, data = [%v]", string(data))
	return nil
}
