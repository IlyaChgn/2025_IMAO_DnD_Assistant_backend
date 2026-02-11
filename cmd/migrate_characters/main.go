package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/models"
	"github.com/IlyaChgn/2025_IMAO_DnD_Assistant_backend/internal/pkg/character/converter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	dryRun      = flag.Bool("dry-run", true, "Run without writing to database")
	verbose     = flag.Bool("verbose", false, "Print detailed output for each character")
	mongoURI    = flag.String("mongo", "", "MongoDB connection URI (or set MONGODB env var)")
	dbName      = flag.String("db", "bestiary_db", "MongoDB database name")
	characterID = flag.String("id", "", "Migrate single character by ObjectID")
)

type migrationStats struct {
	Total     int
	Converted int
	Written   int
	Errors    int
	Warnings  int
}

func main() {
	flag.Parse()

	uri := *mongoURI
	if uri == "" {
		uri = os.Getenv("MONGODB")
	}
	if uri == "" {
		log.Fatal("MongoDB URI required: use -mongo flag or MONGODB env var")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("Connect error:", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database(*dbName)
	srcColl := db.Collection("characters")
	dstColl := db.Collection("characters_v2")

	// Build query
	filter := bson.M{}
	if *characterID != "" {
		oid, err := primitive.ObjectIDFromHex(*characterID)
		if err != nil {
			log.Fatal("Invalid character ID:", err)
		}
		filter["_id"] = oid
	}

	cursor, err := srcColl.Find(ctx, filter)
	if err != nil {
		log.Fatal("Find error:", err)
	}
	defer cursor.Close(ctx)

	stats := &migrationStats{}

	for cursor.Next(ctx) {
		var rawDoc bson.M
		if err := cursor.Decode(&rawDoc); err != nil {
			log.Printf("Decode error: %v", err)
			stats.Errors++
			continue
		}

		stats.Total++

		// MongoDB stores `data` as a nested BSON document, but ConvertLSS expects
		// the LSS file format where `data` is a JSON string (double-encoded).
		// Re-encode the data sub-document as a JSON string inside the envelope.
		if dataDoc, ok := rawDoc["data"]; ok {
			dataJSON, err := json.Marshal(dataDoc)
			if err != nil {
				log.Printf("Marshal data error for %v: %v", rawDoc["_id"], err)
				stats.Errors++
				continue
			}
			rawDoc["data"] = string(dataJSON)
		}

		rawJSON, err := json.Marshal(rawDoc)
		if err != nil {
			log.Printf("Marshal error: %v", err)
			stats.Errors++
			continue
		}

		// Extract userID from the original document
		userID, _ := rawDoc["userID"].(string)
		if userID == "" {
			userID = "unknown"
		}

		// Run conversion
		char, report, err := converter.ConvertLSS(rawJSON, userID)
		if err != nil {
			log.Printf("Conversion error for document %v: %v", rawDoc["_id"], err)
			stats.Errors++
			continue
		}

		stats.Converted++
		stats.Warnings += len(report.Warnings)

		// Preserve the original document ID as a reference
		if origID, ok := rawDoc["_id"].(primitive.ObjectID); ok {
			if char.ImportSource != nil {
				char.ImportSource.Warnings = append(char.ImportSource.Warnings,
					fmt.Sprintf("migrated from characters collection, original _id: %s", origID.Hex()))
			}
		}

		if *verbose || *characterID != "" {
			printCharacterReport(char, report)
		}

		// Write to characters_v2
		if !*dryRun {
			// Upsert by userId + name + importSource reference (idempotent, safe for same-name chars)
			upsertFilter := bson.M{
				"userId":                       char.UserID,
				"name":                         char.Name,
				"importSource.format":          "lss_v2",
			}

			update := bson.M{"$set": char}
			opts := options.Update().SetUpsert(true)

			_, err := dstColl.UpdateOne(ctx, upsertFilter, update, opts)
			if err != nil {
				log.Printf("Write error for %s: %v", char.Name, err)
				stats.Errors++
			} else {
				stats.Written++
			}
		}
	}

	printStats(stats)

	if *dryRun {
		fmt.Println("\nDRY RUN - no changes written. Use -dry-run=false to apply.")
	}
}

func printCharacterReport(char *models.CharacterBase, report *models.ConversionReport) {
	fmt.Printf("\n=== %s ===\n", char.Name)

	if len(char.Classes) > 0 {
		fmt.Printf("  Class: %s (level %d)\n", char.Classes[0].ClassName, char.Classes[0].Level)
	}
	fmt.Printf("  Race: %s\n", char.Race)
	fmt.Printf("  Abilities: STR=%d DEX=%d CON=%d INT=%d WIS=%d CHA=%d\n",
		char.AbilityScores.Str, char.AbilityScores.Dex, char.AbilityScores.Con,
		char.AbilityScores.Int, char.AbilityScores.Wis, char.AbilityScores.Cha)

	if char.HitPoints.MaxOverride != nil {
		fmt.Printf("  HP: %d\n", *char.HitPoints.MaxOverride)
	}
	if char.ArmorClassOverride != nil {
		fmt.Printf("  AC: %d\n", *char.ArmorClassOverride)
	}
	fmt.Printf("  Speed: %d\n", char.BaseSpeed)

	if char.Spellcasting != nil {
		fmt.Printf("  Spellcasting: %s\n", char.Spellcasting.Ability)
		if len(char.Spellcasting.SpellTexts) > 0 {
			fmt.Printf("  Spell text levels: ")
			for level := range char.Spellcasting.SpellTexts {
				fmt.Printf("%d ", level)
			}
			fmt.Println()
		}
	}

	fmt.Printf("  Weapons: %d\n", len(char.Weapons))
	fmt.Printf("  Report: copied=%d parsed=%d skipped=%d warnings=%d\n",
		report.FieldsCopied, report.FieldsParsed, report.FieldsSkipped, len(report.Warnings))

	for _, w := range report.Warnings {
		fmt.Printf("    [%s] %s: %s\n", w.Level, w.Field, w.Message)
	}
}

func printStats(stats *migrationStats) {
	fmt.Println("\n========== MIGRATION STATS ==========")
	fmt.Printf("Total documents:   %d\n", stats.Total)
	fmt.Printf("Converted:         %d\n", stats.Converted)
	fmt.Printf("Written to DB:     %d\n", stats.Written)
	fmt.Printf("Errors:            %d\n", stats.Errors)
	fmt.Printf("Total warnings:    %d\n", stats.Warnings)
}
