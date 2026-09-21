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

	l "github.com/k9io/jsonair/internal/logger"

	"github.com/gin-gonic/gin"
)

type deleteRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func deleteConfig(c *gin.Context) {

	cl := c.MustGet("claims").(*claims)

	req := &deleteRequest{}

	err := decodeStrict(c.Request.Body, req)

	if err == nil {
		err = validateTypeName(req.Type, req.Name)
	}

	if err != nil {

		l.Logger(l.WARN, "Rejected DELETE from %s [%s]: %v", c.ClientIP(), cl.UUID, err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return

	}

	if !scopeAllows(cl.AllowedTypes, req.Type) || !scopeAllows(cl.AllowedNames, req.Name) {

		l.Logger(l.WARN, "'%s' [%s] from %s tried to delete '%s/%s' outside of its scope.", cl.ClientName, cl.UUID, c.ClientIP(), req.Type, req.Name)
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not permitted"})
		return

	}

	deleted, err := sqlDeleteConfig(c.Request.Context(), cl.UUID, req.Type, req.Name)

	if err != nil {

		l.Logger(l.ERROR, "Failed to delete '%s/%s' for uuid '%s': %v", req.Type, req.Name, cl.UUID, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return

	}

	if !deleted {

		l.Logger(l.WARN, "'%s' [%s] from %s tried to delete '%s/%s', which does not exist.", cl.ClientName, cl.UUID, c.ClientIP(), req.Type, req.Name)
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return

	}

	l.Logger(l.INFO, "AUDIT deleted '%s/%s' for '%s' [%s] from %s", req.Type, req.Name, cl.ClientName, cl.UUID, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "type": req.Type, "name": req.Name})

}
