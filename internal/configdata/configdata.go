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

/* Package configdata holds the pieces shared by the programs that _write_
   configurations (jsonair-admin and jsonair-write) so that both store data in
   exactly the same form. */

package configdata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	cry "github.com/k9io/jsonair/internal/crypto"

	"github.com/goccy/go-yaml"
)

// ErrUnknownFormat is returned by Validate for a format other than json, xml or yaml.
var ErrUnknownFormat = errors.New("unknown format — use json, xml, or yaml")

// Validate checks that data is syntactically valid for the given format
// ("json", "xml" or "yaml").
func Validate(format string, data []byte) error {

	switch format {

	case "json":

		var v any
		return json.Unmarshal(data, &v)

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
				return err
			}

			if _, ok := tok.(xml.StartElement); ok {
				hasElement = true
			}
		}

		if !hasElement {
			return fmt.Errorf("not valid XML: no root element found")
		}

		return nil

	case "yaml":

		var v any
		return yaml.Unmarshal(data, &v)

	}

	return ErrUnknownFormat

}

// Encode base64-encodes the raw configuration text and encrypts it, producing
// the value stored in the `config_data` column.
func Encode(rawConfig string, key []byte) (string, error) {

	b64 := base64.StdEncoding.EncodeToString([]byte(rawConfig))

	return cry.Encrypt([]byte(b64), key)

}
