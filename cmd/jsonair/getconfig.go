/**
 ** Copyright (C) 2026 Key9, Inc <k9.io>
 ** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
 **
 ** This file is part of the JSONAir.
 **
 ** This source code is licensed under the MIT license found in the
 ** LICENSE file in the root directory of this source tree.
 **
 **/

package main

import (
	"encoding/base64"
	"net/http"
	//	"fmt"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func GetConfig(c *gin.Context) {

	var err error
	var name string
	var jtype string
	var config_json string

	uuid := c.GetString("uuid")
	client_name, _ := c.Get("client_name")

	jsondata, _ := c.GetRawData()
	jsondata_s := string(jsondata)

	name, jtype, err = GetConfigName(c, jsondata_s)

	if err != nil {

		l.Logger(l.WARN, "%v [%s]", err, c.ClientIP())
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return

	}

	ecode := gjson.Get(jsondata_s, "encode").Bool()

	l.Logger(l.INFO, "%s requested 'config' for '%s/%s' for '%s' [%s]", c.ClientIP(), jtype, name, client_name, uuid)

	config_json, err = SQL_GetConfig(uuid, name, jtype)

	if err != nil {

		l.Logger(l.WARN, "%v", err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return

	}

	if ecode == false {

		c.String(http.StatusOK, config_json)

	} else {

		b64 := base64.StdEncoding.EncodeToString([]byte(config_json))

		config_b64, _ := sjson.Set("", "config", b64)

		c.String(http.StatusOK, config_b64)

	}

}
