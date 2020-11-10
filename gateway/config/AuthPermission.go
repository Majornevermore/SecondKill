package config

import (
	"regexp"
	"strings"
)

var (
	AuthPermitConfig AuthPermitAll
)


type AuthPermitAll struct {
	PermitALL []interface{}
}

func Match(str string) bool {
	if len(AuthPermitConfig.PermitALL) > 0 {
		targetValue := AuthPermitConfig.PermitALL
		for i:=0; i<len(targetValue); i++ {
			s := targetValue[i].(string)
			res, _ := regexp.MatchString(strings.ReplaceAll(s, "**", "(.*?)"), str)
			if res {
				return true
			}
		}
	}
	return false
}
