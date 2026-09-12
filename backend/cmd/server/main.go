package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jchen0102/grok-build/internal/store"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if err := store.Migrate(databaseURL); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	fmt.Println("migrate ok")
}
