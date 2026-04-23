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

	"encoding/json"
	"net/http"

	"github.com/k9io/jsonair/internal/define"
	"github.com/k9io/jsonair/internal/http_req"

	l "github.com/k9io/jsonair/internal/logger"
)

type JWT_Struct struct {
	Access_Token string
}

func PAT_Auth() string {

	var err error
	var JWT *JWT_Struct

	pat_json := fmt.Sprintf(`{"token": "%s"}`, Env.JSONAIR_PAT)
	auth_url := fmt.Sprintf("%s/api/%s/jsonair/auth/token", Env.JSONAIR_URL, define.VERSION)

	results, status_code := http_req.HTTP(pat_json, auth_url, "POST", "")

	if status_code != http.StatusOK {

		l.Logger(l.ERROR, "Error getting Bearer Token.  HTTP Status: %v", status_code)
		os.Exit(1)

	}

	err = json.Unmarshal([]byte(results), &JWT)

	if err != nil {

		l.Logger(l.ERROR, "Cannot parse Bearer Token: %v", err)
		os.Exit(1)

	}

	if JWT.Access_Token == "" {

		l.Logger(l.ERROR, "Unable to find the 'access_token'.")
		os.Exit(1)

	}

	return JWT.Access_Token

}
