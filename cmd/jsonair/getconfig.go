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

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func GetConfig(c *gin.Context) {

	var err error
	var name string
	var jtype string
	var config_data string

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

	decode := gjson.Get(jsondata_s, "decode").Bool()

	l.Logger(l.INFO, "%s requested 'config' for '%s/%s' for '%s' [%s]", c.ClientIP(), jtype, name, client_name, uuid)

	config_data, err = SQL_GetConfig(uuid, name, jtype)

	if err != nil {

		l.Logger(l.WARN, "%v", err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return

	}

	/* Does the client want the configuration in Base64 or not */

	if decode == false {

		c.String(http.StatusOK, config_data)

	} else {

		decode_config_out, err := base64.StdEncoding.DecodeString(config_data)

		if err != nil {
			l.Logger(l.ERROR, "Error decoding base64: %v", err)
			return
		}

		c.String(http.StatusOK, string(decode_config_out))
	}

}
