package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"oauth2-example/handlers"
)

func main() {
	// We create a simple server using http.Server and run.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":" + port),
		Handler: handlers.New(),
	}

	log.Printf("Starting HTTP Server. Listening at %q", server.Addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("%v", err)
	} else {
		log.Println("Server closed!")
	}
}
