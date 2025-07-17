package main

import (
	"log"
	"os"

	"github.com/webpage-analyser-server/internal/app"
	"github.com/webpage-analyser-server/internal/constants"
)

func main() {
	
	env := os.Getenv(constants.EnvAppEnv)
	if env == "" {
		env = constants.EnvDevelopment
	}

	
	application, err := app.New(constants.DefaultConfigDir, env)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	
	if err := application.Start(); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	
	application.WaitForSignal()

	
	if err := application.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
} 