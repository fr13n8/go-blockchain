package main

import (
	"flag"
	"fmt"
)

func main() {
	port := flag.Uint("port", 8080, "port to listen on")
	gateway := flag.String("gateway", "http://localhost:5050", "gateway to connect to blockchain")
	flag.Parse()

	cfg := ServerConfig{
		port:       uint16(*port),
		gateway:    *gateway,
		serverName: "Client app",
		host:       "0.0.0.0",
	}
	s := NewServer(&cfg)
	quit := s.Run()
	<-quit
	fmt.Println("Shutting down client server...")
	s.ShutdownGracefully()
}
