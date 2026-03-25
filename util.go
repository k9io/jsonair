package main

import (
	"fmt"
	"regexp"

	//	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func Remove_Unwanted(str string) string {

	reg := regexp.MustCompile(`[^a-zA-Z0-9-_.]+`)
	return reg.ReplaceAllString(str, "")

}

func GetConfigName(c *gin.Context, jsondata_s string) (string, error) {

	config_name := gjson.Get(jsondata_s, "config").String()
	config_name = Remove_Unwanted(config_name)

	if config_name == "" {

		return "", fmt.Errorf("No 'config' specified in POST request from %s.", c.ClientIP())
	}

	return config_name, nil

}
