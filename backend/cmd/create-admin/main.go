package main

import (
	"backend/internal/database"
	"backend/internal/provision"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	email := flag.String("email", "", "Owner email")
	name := flag.String("name", "TKDHub Admin", "Owner display name")
	org := flag.String("organization", "TKDHub", "Organization name")
	flag.Parse()
	if *email == "" {
		log.Fatal("use -email and supply the password on standard input (scripts/create-admin.sh)")
	}
	password, err := io.ReadAll(io.LimitReader(os.Stdin, 75))
	if err != nil {
		log.Fatal("could not read password")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, database.ConnectionString())
	if err != nil {
		log.Fatal("invalid database configuration")
	}
	defer pool.Close()
	id, err := provision.CreateAdmin(ctx, pool, *name, *email, strings.TrimRight(string(password), "\r\n"), *org)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			log.Fatal("account already exists; no password or permissions were changed")
		}
		log.Fatal("could not provision account: ", err)
	}
	fmt.Printf("Admin created: %s\nOrganization: %s\n", *email, id)
}
