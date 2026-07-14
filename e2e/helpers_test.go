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
	"context"
	"net/http"
	"testing"

	_ "github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
	"github.com/tstangenberg/stratum/internal/server"
)

func mustServerHandler(t *testing.T, srv *server.StratumServer) http.Handler {
	t.Helper()
	if err := srv.Initialize(context.Background()); err != nil {
		t.Fatalf("server.Initialize: %v", err)
	}
	h, err := server.Handler(srv)
	if err != nil {
		t.Fatalf("server.Handler: %v", err)
	}
	return h
}
