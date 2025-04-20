package main

import (
	"flag"
	migrator "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/db"
	"github.com/joho/godotenv"
	"log"

	app "github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/server"
)

var productionFlag = flag.Bool("prod", false, "Run in production mode")
var migrationsFlag = flag.String("migrate", "latest", "Run in migrations mode")

func main() {
	var err error

	flag.Parse()

	if *productionFlag {
		log.Println("Running in production mode")
		err = godotenv.Load("prod.env")
	} else {
		log.Println("Running in development mode")
		err = godotenv.Load(".env")
	}

	migrator.ApplyMigrations(*migrationsFlag)

	if err != nil {
		log.Fatal("Error loading env file ", err)
	}

	srv := new(app.Server)

	if err := srv.Run(); err != nil {
		log.Fatal("Error occurred while starting server ", err)
	}
}
