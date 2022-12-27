package main

import (
	"flag"
	"log"

	clientServer "github.com/fr13n8/go-blockchain/pkg/client/server"
)

func main() {
	// get gateway from flag
	gateway := flag.String("gateway", ":5050", "gateway address")
	flag.Parse()

	cfg := clientServer.Config{
		Port:       8080,
		ServerName: "Wallet",
		Host:       "0.0.0.0",
		Gateway:    "0.0.0.0" + *gateway,
	}
	s := clientServer.NewServer(&cfg)
	log.Printf("[WALLET] Start wallet listen on port %d\n", 8080)
	quit := s.Run()
	<-quit
	log.Printf("[WALLET] Stop wallet listen on port %d\n", 8080)
	s.ShutdownGracefully()
}
