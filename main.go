package main

/* Notes:

   Droppriv needed

*/

import (
	//        "fmt"
	"io/ioutil"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {

	Logger(INFO, "Starting JSONAir.")

	LoadEnv() /* Load environment variables */

	/* Enable remote logging, if needed */

	if Env.SYSLOG_HOST != "" {
		Init_Logger(Env.SYSLOG_HOST, Env.SYSLOG_PROTO)
	}

	SQL_Connect()

	if Env.HTTP_MODE == "production" || Env.HTTP_MODE == "release" {

		//                if debug.X.Load {
		//                       logger.Syslog("debug", "[DEBUG_LOAD] Suppressing Logging.")
		//                }

		gin.SetMode("release")

		gin.DefaultWriter = ioutil.Discard

	} else {

		gin.SetMode(Env.HTTP_MODE)

	}

	router := gin.Default()

	if Env.HTTP_MODE != "production" && Env.HTTP_MODE != "release" {
		router.Use(HTTP_Logger())
	}

	router.Use(Authenticate())

	router.POST("/config", GetConfig)
	router.POST("/debug", GetDebug)
	router.POST("/reload", GetReload)

	if Env.HTTP_TLS == true {

		Logger(INFO, "JSONAir is up and listening for TLS traffic on %s.", Env.HTTP_LISTEN)

		err := router.RunTLS(Env.HTTP_LISTEN, Env.HTTP_CERT, Env.HTTP_KEY)

		if err != nil {

			Logger(ERROR, "Cannot bind to %s or cannot open %s or %s [%v].", Env.HTTP_LISTEN, Env.HTTP_CERT, Env.HTTP_KEY, err)
			os.Exit(1)

		}

	} else {

		Logger(INFO, "JSONAir is up and listening for traffic on %s.", Env.HTTP_LISTEN)

		err := router.Run(Env.HTTP_LISTEN)

		if err != nil {

			Logger(ERROR, "Cannot bind to %s [%v].", Env.HTTP_LISTEN, err)
			os.Exit(1)

		}
	}
}
