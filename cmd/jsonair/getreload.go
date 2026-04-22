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
	"net/http"

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/sjson"
)

func getReload(c *gin.Context) {

	var err error
	var name string
	var jtype string
	var reload string

	uuid := c.GetString("uuid")
	clientName, _ := c.Get("client_name")

	jsondata, err := c.GetRawData()

	if err != nil {
		l.Logger(l.WARN, "Failed to read request body from %s: %v", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	jsonStr := string(jsondata)

	c.Header("Content-Type", "application/json; charset=utf-8")

	name, jtype, err = getConfigName(c, jsonStr)

	if err != nil {

		l.Logger(l.WARN, "%v [%s]", err, c.ClientIP())
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()
		return

	}

	l.Logger(l.INFO, "%s requested 'reload' for '%s/%s' for '%s' [%s]", c.ClientIP(), jtype, name, clientName, uuid)

	reload, err = sqlGetSimple(c.Request.Context(), uuid, name, jtype, "reload")

	if err != nil {

		l.Logger(l.WARN, "%v", err)
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()
		return

	}

	reloadJSON, _ := sjson.Set("", "reload", reload)

	c.String(http.StatusOK, reloadJSON)

}
