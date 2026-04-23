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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPATAuth_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		if body["token"] != "my-test-pat" {
			t.Errorf("token = %q, want %q", body["token"], "my-test-pat")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"access_token": "bearer-xyz"})
	}))
	defer srv.Close()

	Env.JSONAIR_PAT = "my-test-pat"
	Env.JSONAIR_URL = srv.URL

	token := PAT_Auth()
	if token != "bearer-xyz" {
		t.Errorf("PAT_Auth() = %q, want %q", token, "bearer-xyz")
	}
}

func TestPATAuth_RequestUsesJSONMarshal(t *testing.T) {
	specialPAT := `has"quote&special<chars>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body is not valid JSON (json.Marshal fix may be missing): %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["token"] != specialPAT {
			t.Errorf("token = %q, want %q", body["token"], specialPAT)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"access_token": "ok-token"})
	}))
	defer srv.Close()

	Env.JSONAIR_PAT = specialPAT
	Env.JSONAIR_URL = srv.URL

	token := PAT_Auth()
	if token != "ok-token" {
		t.Errorf("PAT_Auth() = %q, want %q", token, "ok-token")
	}
}
