/*
** Copyright (C) 2026 Key9, Inc <k9.io>
** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
**
** This file is part of the HighVolt JSON analysis engine
**
** This program is free software: you can redistribute it and/or modify
** it under the terms of the GNU Affero General Public License as published by
** the Free Software Foundation, either version 3 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
** GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License
** along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package http_req

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"time"

	l "github.com/k9io/jsonair/internal/logger"
)

func HTTP(json_data string, url string, http_type string, bearer_token string) (string, int) {

	client := http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest(http_type, url, bytes.NewBuffer([]byte(json_data)))

	if err != nil {

		l.Logger(l.ERROR, "Unable to establish API connection: %v", err)
		os.Exit(1)

	}

	if bearer_token != "" {

		req.Header.Set("Authorization", "Bearer "+bearer_token)

	}

	res, err := client.Do(req)

	if err != nil {

		l.Logger(l.ERROR, "Unable to client.Do(): %v", err)
		os.Exit(1)

	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {

		l.Logger(l.ERROR, "Unable to get body from request: %v", err)
		os.Exit(1)

	}

	return string(body), res.StatusCode

}
