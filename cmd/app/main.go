package main

import (
	"github.com/joho/godotenv"
	"log"

	app "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading env file ", err)
	}

	srv := new(app.Server)

	if err := srv.Run(); err != nil {
		log.Fatal("Error occurred while starting server ", err)
	}
}
