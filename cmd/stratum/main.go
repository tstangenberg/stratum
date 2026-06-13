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
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tstangenberg/stratum/internal/config"
	"github.com/tstangenberg/stratum/internal/plugin"
	dbplugin "github.com/tstangenberg/stratum/internal/plugin/database"
	"github.com/tstangenberg/stratum/internal/server"
)

func resolveAddr() string {
	if addr := os.Getenv("STRATUM_SERVER_ADDR"); addr != "" {
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

func resolveMaxListLimit() int {
	if s := os.Getenv("STRATUM_SERVER_LIST_MAX_LIMIT"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			log.Fatalf("STRATUM_SERVER_LIST_MAX_LIMIT: %v", err)
		}
		return n
	}
	return 0
}

func run(addr string) error {
	pool, plugins := defaultPlugins()
	if pool != nil {
		defer pool.Close()
	}
	srv := server.NewStratumServer(plugins...).WithMaxListLimit(resolveMaxListLimit())
	if pool != nil {
		srv = srv.WithDB(pool)
	}
	return http.ListenAndServe(addr, server.Handler(srv))
}

func defaultPlugins() (*pgxpool.Pool, []plugin.HealthPlugin) {
	dsn := os.Getenv("STRATUM_DATABASE_URL")
	if dsn == "" {
		log.Printf("STRATUM_DATABASE_URL not set; database health check and schema operations disabled")
		return nil, nil
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Printf("failed to create pgxpool: %v", err)
		return nil, nil
	}
	return pool, []plugin.HealthPlugin{dbplugin.New(pool)}
}
