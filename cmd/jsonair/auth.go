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
	"strings"

	"github.com/gin-gonic/gin"
)

func Authenticate() gin.HandlerFunc {

	var err error

	return func(c *gin.Context) {

		full_header := c.GetHeader(Env.AUTH_HEADER)

		/* No key given,  return with error */

		if full_header == "" {
			Logger(WARN, "Authentication failed for [NO KEY] [%s]", c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"error": "authentication failed"})
			c.Abort()
			return
		}

		temp_value := strings.Split(full_header, ":")

		/* Validate the string properly split */

		if len(temp_value) != 2 {
			Logger(WARN, "Authentication failed for [INVALID KEY] [%s]", c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"error": "api authentication failed"})
			c.Abort()
			return
		}

		/* Assign more sane values */

		uuid := Remove_Unwanted(temp_value[0])
		key := Remove_Unwanted(temp_value[1])

		auth_check := SQL_Auth(uuid, key)

		if auth_check == false {

			Logger(WARN, "Authentication failed for %s [%s]", uuid, c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"error": "authentication failed"})
			c.Abort()
			return

		}

		err = SQL_Update_Last_Login(uuid, key)

		if err != nil {

			Logger(ERROR, "Error updating 'last_login': %v", err)

		}

		c.Set("uuid", uuid)

		Logger(NOTICE, "%s successfully authenticated. [%s]", uuid, c.ClientIP())

	}

}
