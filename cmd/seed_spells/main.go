package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/spells/seed"
)

// Command line flags
var (
	dryRun   = flag.Bool("dry-run", true, "Run without writing to database (default: true)")
	mongoURI = flag.String("mongo", "", "MongoDB connection URI (or set MONGODB env var)")
	verbose  = flag.Bool("verbose", false, "Print each spell as it's processed")
)

// SeedStats tracks seeding progress.
type SeedStats struct {
	Total    int
	Inserted int
	Updated  int
	Skipped  int
	Errors   int
}

func main() {
	flag.Parse()

	// Parse spell data from shared seed package (single source of truth)
	var spells []bson.M
	if err := json.Unmarshal(seed.SRDSpellsJSON(), &spells); err != nil {
		log.Fatal("Failed to parse spells JSON:", err)
	}

	fmt.Printf("Loaded %d spell definitions from seed package\n", len(spells))

	// Resolve MongoDB URI
	uri := *mongoURI
	if uri == "" {
		uri = os.Getenv("MONGODB")
	}
	if uri == "" {
		// In dry-run mode, allow running without a DB connection for validation
		if *dryRun {
			fmt.Println("\n--- DRY RUN (no DB connection) ---")
			printSpellSummary(spells)
			fmt.Println("\n✅ JSON parsed successfully. Use -mongo flag or MONGODB env to connect.")
			return
		}
		log.Fatal("MongoDB URI required: use -mongo flag or MONGODB env var")
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Connect error:", err)
	}
	defer client.Disconnect(ctx)

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Ping error:", err)
	}
	fmt.Println("Connected to MongoDB")

	coll := client.Database("bestiary_db").Collection("spell_definitions")

	stats := &SeedStats{}

	for _, spell := range spells {
		engName, _ := spell["engName"].(string)
		if engName == "" {
			log.Printf("Warning: spell missing engName, skipping")
			stats.Errors++
			continue
		}

		name := ""
		if n, ok := spell["name"].(map[string]interface{}); ok {
			name, _ = n["eng"].(string)
		}

		stats.Total++

		if *verbose {
			level, _ := spell["level"].(float64)
			fmt.Printf("  [%d] %-25s (level %d)\n", stats.Total, name, int(level))
		}

		if *dryRun {
			stats.Skipped++
			continue
		}

		// Upsert by engName — idempotent
		filter := bson.M{"engName": engName}
		update := bson.M{"$set": spell}
		opts := options.Update().SetUpsert(true)

		result, err := coll.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Error upserting %s: %v", engName, err)
			stats.Errors++
			continue
		}

		if result.UpsertedCount > 0 {
			stats.Inserted++
		} else if result.ModifiedCount > 0 {
			stats.Updated++
		} else {
			stats.Skipped++
		}
	}

	printStats(stats)

	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN — no changes written. Use -dry-run=false to apply.")
	}
}

func printSpellSummary(spells []bson.M) {
	levelCounts := make(map[int]int)
	for _, spell := range spells {
		level := 0
		if l, ok := spell["level"].(float64); ok {
			level = int(l)
		}
		levelCounts[level]++

		if *verbose {
			engName, _ := spell["engName"].(string)
			name := ""
			if n, ok := spell["name"].(map[string]interface{}); ok {
				name, _ = n["eng"].(string)
			}
			fmt.Printf("  %-25s (level %d) [%s]\n", name, level, engName)
		}
	}

	fmt.Printf("\nSpells by level:\n")
	for level := 0; level <= 9; level++ {
		if count, ok := levelCounts[level]; ok {
			label := fmt.Sprintf("Level %d", level)
			if level == 0 {
				label = "Cantrips"
			}
			fmt.Printf("  %-12s %d\n", label, count)
		}
	}
}

func printStats(stats *SeedStats) {
	fmt.Println("\n========== SEED STATS ==========")
	fmt.Printf("Total processed:  %d\n", stats.Total)
	fmt.Printf("Inserted:         %d\n", stats.Inserted)
	fmt.Printf("Updated:          %d\n", stats.Updated)
	fmt.Printf("Skipped (no-op):  %d\n", stats.Skipped)
	fmt.Printf("Errors:           %d\n", stats.Errors)
}
