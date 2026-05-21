// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"log"
	"net/http"

	"github.com/tstangenberg/stratum/internal/server"
)

func main() {
	if err := run(":8080"); err != nil {
		log.Fatal(err)
	}
}

func run(addr string) error {
	srv := server.NewStratumServer()
	return http.ListenAndServe(addr, server.Handler(srv))
}
