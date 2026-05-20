// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"log"
	"net/http"

	"github.com/tstangenberg/stratum/internal/server"
)

func main() {
	srv := server.NewStratumServer()
	h := server.Handler(srv)
	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", h); err != nil {
		log.Fatal(err)
	}
}
