package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	infrastructurepostgres "example.com/taskservice/internal/infrastructure/postgres"
	postgresrepo "example.com/taskservice/internal/repository/postgres"
	"example.com/taskservice/internal/scheduler"
)

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := infrastructurepostgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer pool.Close()

	repo := postgresrepo.New(pool)
	sched := scheduler.New(repo)

	// parse flags: --days N or --month
	var days int
	var month bool
	flag.IntVar(&days, "days", 0, "number of days ahead to create occurrences for (overrides --month)")
	flag.BoolVar(&month, "month", false, "create occurrences for the remainder of the current month")
	flag.Parse()

	if month && days == 0 {
		// compute days until end of month (inclusive)
		now := time.Now().UTC()
		firstOfNextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		days = int(firstOfNextMonth.Sub(dateOnly(now)).Hours() / 24)
	}

	if days <= 0 {
		days = 1
	}

	created, err := sched.Run(ctx, days)
	if err != nil {
		log.Fatalf("scheduler run: %v", err)
	}

	fmt.Printf("scheduler: created %d occurrences\n", created)
}

type config struct {
	DatabaseDSN string
}

func loadConfig() config {
	return config{
		DatabaseDSN: envOrDefault("DATABASE_DSN", "postgres://postgres:postgres@localhost:5432/taskservice?sslmode=disable"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dateOnly duplicates scheduler.dateOnly because cmd package is separate
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
