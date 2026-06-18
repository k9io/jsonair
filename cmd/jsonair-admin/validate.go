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
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

func validateConfig(c *gin.Context) {

	format := c.PostForm("format")
	data := []byte(c.PostForm("data"))

	if len(data) == 0 {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": "no data provided"})
		return
	}

	var parseErr error

	switch format {

	case "json":
		var v any
		parseErr = json.Unmarshal(data, &v)

	case "xml":
		// Walk all tokens to catch syntax errors, and require at least one
		// start element — Go's decoder accepts bare text without elements,
		// which is not a valid XML document.
		decoder := xml.NewDecoder(bytes.NewReader(data))
		hasElement := false
		for {
			tok, err := decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				parseErr = err
				break
			}
			if _, ok := tok.(xml.StartElement); ok {
				hasElement = true
			}
		}
		if parseErr == nil && !hasElement {
			parseErr = fmt.Errorf("not valid XML: no root element found")
		}

	case "yaml":
		var v any
		parseErr = yaml.Unmarshal(data, &v)

	default:
		c.JSON(http.StatusBadRequest, gin.H{"valid": false, "error": "unknown format — use json, xml, or yaml"})
		return

	}

	if parseErr != nil {
		c.JSON(http.StatusOK, gin.H{"valid": false, "error": parseErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": true, "error": ""})

}
