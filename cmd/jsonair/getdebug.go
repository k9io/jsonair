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

func GetDebug(c *gin.Context) {

	var err error
	var name string
	var jtype string
	var debug string

	uuid := c.MustGet("uuid").(string) /* gin will panic if this isn't there (as it should) */

	jsondata, _ := c.GetRawData()
	jsondata_s := string(jsondata)

	c.Header("Content-Type", "application/json; charset=utf-8")

	name, jtype, err = GetConfigName(c, jsondata_s)

	if err != nil {

		l.Logger(l.WARN, "%v [%s]", err, c.ClientIP())
		c.String(http.StatusNotFound, `{"status":"not found","code":404}`)
		c.Abort()
		return

	}

	l.Logger(l.INFO, "%s requested debug for %s", c.ClientIP(), name)

	debug, err = SQL_GetSimple(uuid, name, jtype, "debug")

	if err != nil {

		l.Logger(l.WARN, "%v", err)

		status := `{"status":"not found","code":404}`
		c.String(http.StatusNotFound, status)
		c.Abort()

		return

	}

	debug_json, _ := sjson.Set("", "debug", debug)

	c.String(http.StatusOK, debug_json)

}
