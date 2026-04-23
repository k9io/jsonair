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

func getConfig(c *gin.Context) {

	var err error
	var name string
	var jtype string
	var configData string

	uuid := c.GetString("uuid")
	clientName := c.GetString("client_name")

	jsondata, err := c.GetRawData()

	if err != nil {
		l.Logger(l.WARN, "Failed to read request body from %s: %v", c.ClientIP(), err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	jsonStr := string(jsondata)

	name, jtype, err = getConfigName(c, jsonStr)

	if err != nil {

		l.Logger(l.WARN, "%v [%s]", err, c.ClientIP())
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return

	}

	decode := gjson.Get(jsonStr, "decode").Bool()

	l.Logger(l.INFO, "%s requested 'config' for '%s/%s' for '%s' [%s]", c.ClientIP(), jtype, name, clientName, uuid)

	configData, err = sqlGetConfig(c.Request.Context(), uuid, name, jtype)

	if err != nil {

		l.Logger(l.WARN, "%v", err)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return

	}

	/* Does the client want the configuration in Base64 or not */

	if !decode {

		c.String(http.StatusOK, configData)

	} else {

		decoded, err := base64.StdEncoding.DecodeString(configData)

		if err != nil {
			l.Logger(l.ERROR, "Error decoding base64: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode configuration"})
			return
		}

		c.String(http.StatusOK, string(decoded))
	}

}
