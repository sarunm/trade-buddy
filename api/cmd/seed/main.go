package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"trade-buddy/api/internal/db"
)

func main() {
	sqlitePath := flag.String("sqlite", "data/journal.db", "path to Python SQLite journal.db")
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres connection string")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("DSN is required (via --dsn or DATABASE_URL env)")
	}

	// 1. Open SQLite
	sqldb, err := sql.Open("sqlite", *sqlitePath)
	if err != nil {
		log.Fatalf("failed to open sqlite: %v", err)
	}
	defer sqldb.Close()

	// 2. Open Postgres
	pgdb, err := db.Connect(*dsn)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	// 3. Query SQLite setups + LEFT JOIN outcomes
	query := `
		SELECT s.id, s.created_at, s.symbol, s.timeframe, s.direction,
		       s.entry, s.stop_loss, s.take_profit, s.risk_reward, s.confidence,
		       o.id AS outcome_id, o.result, o.r_multiple
		FROM setups s
		LEFT JOIN outcomes o ON o.setup_id = s.id
	`
	rows, err := sqldb.Query(query)
	if err != nil {
		log.Fatalf("failed to query sqlite: %v", err)
	}
	defer rows.Close()

	alertsCount := 0
	outcomesCount := 0

	for rows.Next() {
		var (
			id, createdAtStr, symbol, timeframe, direction string
			entry, stopLoss, takeProfit, riskReward, confidence sql.NullFloat64
			outcomeID, result sql.NullString
			rMultiple sql.NullFloat64
		)

		err := rows.Scan(
			&id, &createdAtStr, &symbol, &timeframe, &direction,
			&entry, &stopLoss, &takeProfit, &riskReward, &confidence,
			&outcomeID, &result, &rMultiple,
		)
		if err != nil {
			log.Printf("failed to scan row: %v", err)
			continue
		}

		// Build Postgres alert ID
		alertUUID, err := uuid.Parse(id)
		if err != nil {
			log.Printf("failed to parse uuid %s: %v", id, err)
			continue
		}

		createdAt := parseTime(createdAtStr)

		// context JSONB: {"pattern": direction, "session": "unknown", "timeframe": timeframe, "detector_version": "python-seed"}
		contextMap := map[string]interface{}{
			"pattern":          direction,
			"session":          "unknown",
			"timeframe":        timeframe,
			"detector_version": "python-seed",
		}
		contextJSON, _ := json.Marshal(contextMap)

		status := "open"
		if outcomeID.Valid {
			status = "resolved"
		}

		// Build alerts INSERT
		// direction is required (NOT NULL) in Postgres schema
		err = pgdb.Exec(`
			INSERT INTO alerts (id, symbol, timeframe, direction, entry, stop_loss, take_profit, risk_reward, confidence, context, status, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?::jsonb, ?, ?)
			ON CONFLICT (id) DO NOTHING`,
			alertUUID, symbol, timeframe, direction, entry, stopLoss, takeProfit, riskReward, confidence, string(contextJSON), status, createdAt,
		).Error
		if err != nil {
			log.Printf("failed to insert alert %s: %v", alertUUID, err)
			continue
		}
		alertsCount++

		// If outcome exists, build alert_outcomes INSERT
		if outcomeID.Valid {
			detailsMap := map[string]interface{}{
				"r_multiple": rMultiple.Float64,
			}
			detailsJSON, _ := json.Marshal(detailsMap)

			// outcome = result ("win"/"loss")
			// details JSONB: {"r_multiple": r_multiple}
			err = pgdb.Exec(`
				INSERT INTO alert_outcomes (alert_id, outcome, details, created_at)
				VALUES (?, ?, ?::jsonb, now())
				ON CONFLICT DO NOTHING`,
				alertUUID, result.String, string(detailsJSON),
			).Error
			if err != nil {
				log.Printf("failed to insert outcome for alert %s: %v", alertUUID, err)
			} else {
				outcomesCount++
				
				// Update alerts.resolved_at if it's resolved and not already set
				pgdb.Exec(`UPDATE alerts SET resolved_at = now() WHERE id = ? AND resolved_at IS NULL`, alertUUID)
			}
		}
	}

	fmt.Printf("Seeded %d alerts, %d outcomes\n", alertsCount, outcomesCount)
}

func parseTime(s string) time.Time {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.999999",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Now()
}
