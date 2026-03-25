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

func GetConfigName(c *gin.Context, jsondata_s string) (string, string, error) {

	jtype := gjson.Get(jsondata_s, "type").String()
	jtype = Remove_Unwanted(jtype)

        if jtype == "" {

                return "", "", fmt.Errorf("No 'type' specified in POST request from %s.", c.ClientIP())
        }

	name := gjson.Get(jsondata_s, "name").String()
	name = Remove_Unwanted(name)

	if name == "" {

		return "", "", fmt.Errorf("No 'name' specified in POST request from %s.", c.ClientIP())
	}

	return name, jtype,  nil

}
