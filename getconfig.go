package main

import (
	//"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	// "github.com/tidwall/gjson"
)

func GetConfig(c *gin.Context) {

	var err error
	var name string
	var jtype string
	var config_json string

	uuid := c.MustGet("uuid").(string) /* gin will panic if this isn't there (as it should) */

	jsondata, _ := c.GetRawData()
	jsondata_s := string(jsondata)

	c.Header("Content-Type", "application/json; charset=utf-8")

	name, jtype, err = GetConfigName(c, jsondata_s)

	if err != nil {

		Logger(WARN, "%v [%s]", err, c.ClientIP())
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()
		return

	}

	Logger(INFO, "%s requested configuration for %s", c.ClientIP(), name)

	config_json, err = SQL_GetConfig(uuid, name, jtype)

	if err != nil {

		Logger(WARN, "%v", err)
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()
		return

	}

	c.String(http.StatusOK, config_json)

}
