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
	"os"
	"os/user"
	"regexp"
	"strconv"
	"syscall"

	l "github.com/k9io/jsonair/internal/logger"

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

func dropPrivileges(username string) {

	currentUID := os.Getuid()

	if currentUID != 0 {
		l.Logger(l.NOTICE, "Not running as root. Not dropping privileges.")
		return
	}

	l.Logger(l.NOTICE, "Dropping privileges to '%s'.", username)

	u, err := user.Lookup(username)

	if err != nil {
		l.Logger(l.ERROR, "User lookup failed: %v", err)
		os.Exit(1)
	}

	uid, _ := strconv.Atoi(u.Uid)
	gid, _ := strconv.Atoi(u.Gid)

	err = syscall.Setgroups([]int{gid})

	if err != nil {
		l.Logger(l.NOTICE, "'setgroups' failed: %v", err)
		os.Exit(1)
	}

	err = syscall.Setgid(gid)

	if err != nil {
		l.Logger(l.NOTICE, "'setgid' failed: %v", err)
		os.Exit(1)
	}

	err = syscall.Setuid(uid)

	if err != nil {
		l.Logger(l.NOTICE, "'setuid' failed: %v", err)
		os.Exit(1)
	}
}
