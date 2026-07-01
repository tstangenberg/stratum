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

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/tstangenberg/stratum/internal/config"
	"github.com/tstangenberg/stratum/internal/plugin"
	_ "github.com/tstangenberg/stratum/internal/plugin/auth/apikey"
	dbplugin "github.com/tstangenberg/stratum/internal/plugin/database"
	_ "github.com/tstangenberg/stratum/internal/plugin/pagination/simple"
	"github.com/tstangenberg/stratum/internal/server"
)

func resolveAddr() string {
	if addr := os.Getenv(config.EnvServerAddr); addr != "" {
		return addr
	}
	return ":8080"
}

func main() {
	if err := config.Load(); err != nil {
		log.Fatal(err)
	}
	addr := resolveAddr()
	log.Printf("listening on %s", addr)
	if err := run(addr); err != nil {
		log.Fatal(err)
	}
}

func run(addr string) error {
	srv := server.NewStratumServer()
	pool := dbplugin.Pool()
	if pool != nil {
		defer dbplugin.ClosePool()
		srv = srv.WithDB(pool)
	} else {
		log.Printf("STRATUM_DATABASE_URL not set; schema operations disabled")
	}
	srv = srv.WithMiddlewares(plugin.BuildMiddlewares()...)
	h, err := server.Handler(srv)
	if err != nil {
		return err
	}
	return http.ListenAndServe(addr, h)
}
