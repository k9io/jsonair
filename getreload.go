package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

func GetReload(c *gin.Context) {

	var err error
	var config_name string
	var reload string

	uuid := c.MustGet("uuid").(string) /* gin will panic if this isn't there (as it should) */

	jsondata, _ := c.GetRawData()
	jsondata_s := string(jsondata)

	c.Header("Content-Type", "application/json; charset=utf-8")

	config_name, err = GetConfigName(c, jsondata_s)

	if err != nil {

		Logger(WARN, "%v [%s]", err, c.ClientIP())
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()
		return

	}

	Logger(INFO, "%s requested reload for %s", c.ClientIP(), config_name)

	reload, err = SQL_GetSimple(uuid, config_name, "reload")

	if err != nil {

		Logger(WARN, "%v", err)
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()

		return

	}

	reload_json, _ := sjson.Set("", "reload", reload)

	c.String(http.StatusOK, reload_json)

}
