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
	"regexp"
	"strings"
)

/* 'type' and 'name' are limited to the characters the read API keeps when it
   sanitizes a request (see removeUnwanted() in cmd/jsonair/util.go).  The read
   API _strips_ everything else, so a name containing other characters could be
   stored but never retrieved.  The writer therefore rejects them outright. */

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9\-_.]+$`)

const (
	maxTypeLen   = 128 /* `configurations`.`type`   */
	maxNameLen   = 127 /* `configurations`.`name`   */
	maxReloadLen = 255 /* `configurations`.`reload` */
	maxDebugLen  = 128 /* `configurations`.`debug`  */
)

func validName(s string, maxLen int) bool {
	return len(s) <= maxLen && validNameRe.MatchString(s)
}

/* scopeAllows reports whether 'value' is permitted by 'list', a comma separated
   list taken from the `write_keys` table.  Each entry is one of:

     *        anything
     prefix*  anything starting with 'prefix'
     exact    exactly 'exact'

   An empty list allows nothing. */

func scopeAllows(list string, value string) bool {

	for _, entry := range strings.Split(list, ",") {

		entry = strings.TrimSpace(entry)

		switch {

		case entry == "":
			continue

		case entry == "*":
			return true

		case strings.HasSuffix(entry, "*"):
			if strings.HasPrefix(value, strings.TrimSuffix(entry, "*")) {
				return true
			}

		case entry == value:
			return true

		}
	}

	return false

}
