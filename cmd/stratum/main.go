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

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tstangenberg/stratum/internal/config"
	"github.com/tstangenberg/stratum/internal/plugin"
	_ "github.com/tstangenberg/stratum/internal/plugin/auth/apikey"
	_ "github.com/tstangenberg/stratum/internal/plugin/database"
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

func run(addr string) error {
	pool := connectDB()
	if pool != nil {
		defer pool.Close()
	}
	srv := server.NewStratumServer()
	if pool != nil {
		srv = srv.WithDB(pool)
	}
	srv = srv.WithMiddlewares(plugin.BuildMiddlewares()...)
	return http.ListenAndServe(addr, server.Handler(srv))
}

func connectDB() *pgxpool.Pool {
	dsn := os.Getenv("STRATUM_DATABASE_URL")
	if dsn == "" {
		log.Printf("STRATUM_DATABASE_URL not set; schema operations disabled")
		return nil
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Printf("failed to create pgxpool: %v", err)
		return nil
	}
	return pool
}
