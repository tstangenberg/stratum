// Copyright (C) 2026 Thorben Stangenberg
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tstangenberg/stratum/internal/server"
)

func TestRootRedirect(t *testing.T) {
	handler := mustServerHandler(t, server.NewStratumServer())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "/ui" {
		t.Fatalf("expected Location=/ui, got %q", loc)
	}
}
