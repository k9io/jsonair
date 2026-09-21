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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/k9io/jsonair/internal/configdata"
	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
)

/* putRequest is the body of PUT /config.

   'config_data' is the _raw_ configuration text (not base64).  jsonair-write
   base64 encodes and encrypts it, exactly as jsonair-admin does.  'format' is
   optional; when set to json, xml or yaml the data is checked before storing.
   'reload' and 'debug' are optional; when omitted they are left unchanged on an
   existing configuration (and empty on a new one). */

type putRequest struct {
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	ConfigData *string `json:"config_data"`
	Format     string  `json:"format"`
	Reload     *string `json:"reload"`
	Debug      *string `json:"debug"`
}

/* decodeStrict decodes a single JSON object, rejecting unknown fields (so a
   typo like "confg_data" is an error rather than a silent no-op) and trailing data. */

func decodeStrict(r io.Reader, v any) error {

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON body: %v", err)
	}

	if dec.More() {
		return errors.New("invalid JSON body: unexpected data after object")
	}

	return nil

}

func validateTypeName(jtype, name string) error {

	if !validName(jtype, maxTypeLen) {
		return fmt.Errorf("'type' is required, at most %d characters, and may only contain a-z A-Z 0-9 - _ .", maxTypeLen)
	}

	if !validName(name, maxNameLen) {
		return fmt.Errorf("'name' is required, at most %d characters, and may only contain a-z A-Z 0-9 - _ .", maxNameLen)
	}

	return nil

}

/* parsePutRequest reads and validates a PUT body.  The returned error text is
   safe to send back to the caller. */

func parsePutRequest(r io.Reader) (*putRequest, error) {

	req := &putRequest{}

	if err := decodeStrict(r, req); err != nil {
		return nil, err
	}

	if err := validateTypeName(req.Type, req.Name); err != nil {
		return nil, err
	}

	if req.ConfigData == nil || *req.ConfigData == "" {
		return nil, errors.New("'config_data' is required")
	}

	if req.Reload != nil && len(*req.Reload) > maxReloadLen {
		return nil, fmt.Errorf("'reload' may be at most %d bytes", maxReloadLen)
	}

	if req.Debug != nil && len(*req.Debug) > maxDebugLen {
		return nil, fmt.Errorf("'debug' may be at most %d bytes", maxDebugLen)
	}

	if req.Format != "" {

		if err := configdata.Validate(req.Format, []byte(*req.ConfigData)); err != nil {
			return nil, fmt.Errorf("'config_data' is not valid %s: %v", req.Format, err)
		}

	}

	return req, nil

}

func putConfig(c *gin.Context) {

	cl := c.MustGet("claims").(*claims)

	req, err := parsePutRequest(c.Request.Body)

	if err != nil {

		l.Logger(l.WARN, "Rejected PUT from %s [%s]: %v", c.ClientIP(), cl.UUID, err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}

	if !scopeAllows(cl.AllowedTypes, req.Type) || !scopeAllows(cl.AllowedNames, req.Name) {

		l.Logger(l.WARN, "'%s' [%s] from %s tried to write '%s/%s' outside of its scope.", cl.ClientName, cl.UUID, c.ClientIP(), req.Type, req.Name)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not permitted"})
		return

	}

	encrypted, err := configdata.Encode(*req.ConfigData, Env.ConfigEncryptKey)

	if err != nil {

		l.Logger(l.ERROR, "Failed to encrypt '%s/%s' for uuid '%s': %v", req.Type, req.Name, cl.UUID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return

	}

	/* The uuid always comes from the authenticated token, never from the request. */

	created, err := sqlUpsertConfig(c.Request.Context(), cl.UUID, req.Type, req.Name, req.Reload, req.Debug, encrypted)

	if err != nil {

		l.Logger(l.ERROR, "Failed to write '%s/%s' for uuid '%s': %v", req.Type, req.Name, cl.UUID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return

	}

	/* Audit trail: who wrote what.  A hash of the content is logged, never the content. */

	sum := sha256.Sum256([]byte(*req.ConfigData))
	status, code := "updated", http.StatusOK

	if created {
		status, code = "created", http.StatusCreated
	}

	l.Logger(l.INFO, "AUDIT %s '%s/%s' for '%s' [%s] from %s (sha256:%s, %d bytes)",
		status, req.Type, req.Name, cl.ClientName, cl.UUID, c.ClientIP(), hex.EncodeToString(sum[:]), len(*req.ConfigData))

	c.JSON(code, gin.H{"status": status, "type": req.Type, "name": req.Name})

}
