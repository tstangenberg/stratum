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
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/tstangenberg/stratum/internal/plugin"
	dbplugin "github.com/tstangenberg/stratum/internal/plugin/database"
	"github.com/tstangenberg/stratum/internal/server"
)

func resolveAddr() string {
	if addr := os.Getenv("STRATUM_ADDR"); addr != "" {
		return addr
	}
	return ":8080"
}

func main() {
	addr := resolveAddr()
	log.Printf("listening on %s", addr)
	if err := run(addr); err != nil {
		log.Fatal(err)
	}
}

func run(addr string) error {
	db, pool, plugins := defaultPlugins()
	if db != nil {
		defer db.Close()
	}
	if pool != nil {
		defer pool.Close()
	}
	srv := server.NewStratumServer(plugins...)
	if pool != nil {
		srv = srv.WithDB(pool)
	}
	return http.ListenAndServe(addr, server.Handler(srv))
}

func defaultPlugins() (*sql.DB, *pgxpool.Pool, []plugin.HealthPlugin) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Printf("DATABASE_URL not set; database health check and schema operations disabled")
		return nil, nil, nil
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Printf("failed to open database: %v", err)
		return nil, nil, nil
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Printf("failed to create pgxpool: %v; schema operations disabled", err)
		return db, nil, []plugin.HealthPlugin{dbplugin.New(db)}
	}
	return db, pool, []plugin.HealthPlugin{dbplugin.New(db)}
}
