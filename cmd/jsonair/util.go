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
	"fmt"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

var sanitizeRe = regexp.MustCompile(`[^a-zA-Z0-9-_.]+`)

func removeUnwanted(str string) string {
	return sanitizeRe.ReplaceAllString(str, "")
}

func getConfigName(c *gin.Context, jsonStr string) (string, string, error) {

	jtype := gjson.Get(jsonStr, "type").String()
	jtype = removeUnwanted(jtype)

	if jtype == "" {
		return "", "", fmt.Errorf("No 'type' specified in POST request from %s.", c.ClientIP())
	}

	name := gjson.Get(jsonStr, "name").String()
	name = removeUnwanted(name)

	if name == "" {
		return "", "", fmt.Errorf("No 'name' specified in POST request from %s.", c.ClientIP())
	}

	return name, jtype, nil

}

