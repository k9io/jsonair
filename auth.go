package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func Authenticate() gin.HandlerFunc {

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

		c.Set("uuid", uuid)

		Logger(NOTICE, "%s successfully authenticated.", uuid)

	}

}
