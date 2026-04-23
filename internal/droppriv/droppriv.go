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

package droppriv

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	l "github.com/k9io/jsonair/internal/logger"

)

func DropPrivileges(username string) {

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
