package main

import (
	"backend/internal/database"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, database.ConnectionString())
	if err != nil {
		log.Fatal("invalid database configuration")
	}
	defer pool.Close()
	if err = database.Migrate(ctx, pool); err != nil {
		log.Fatal(err)
	}
	log.Print("migrations applied")
}
