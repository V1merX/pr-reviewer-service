package main

import (
	"context"
	"log"

	"github.com/V1merX/pr-reviewer-service/internal/app"
)

func main() {
	ctx := context.Background()

	a, err := app.New()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := a.Run(ctx); err != nil {
		log.Fatalf("application terminated: %v", err)
	}
}
