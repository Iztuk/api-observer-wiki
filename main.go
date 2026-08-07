package main

import (
	"api-observer-wiki/internal/views"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	websiteHandler, err := views.NewHandler()
	if err != nil {
		log.Fatal(err.Error())
	}

	mux := http.NewServeMux()

	websiteHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
	}

	fmt.Printf("Server is running on http://localhost%s", server.Addr)
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}
