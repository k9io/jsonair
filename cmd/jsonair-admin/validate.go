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
	"errors"
	"net/http"

	"github.com/k9io/jsonair/internal/configdata"

	"github.com/gin-gonic/gin"
)

func validateConfig(c *gin.Context) {

	format := c.PostForm("format")
	data := []byte(c.PostForm("data"))

	if len(data) == 0 {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "no data provided"})
		return
	}

	err := configdata.Validate(format, data)

	if errors.Is(err, configdata.ErrUnknownFormat) {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": err.Error()})
		return
	}

	if err != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "error": ""})

}
