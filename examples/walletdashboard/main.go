package main

import (
	"log"
	"net/http"
	"os"

	"github.com/jumpcrypto/crosschain/dashboard"
)

func main() {
	addr := os.Getenv("DASHBOARD_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	svc := dashboard.NewService()
	handler := dashboard.NewHandler(svc)

	log.Printf("wallet connect dashboard listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
