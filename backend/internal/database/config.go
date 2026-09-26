package database

import (
	"net"
	"net/url"
	"os"
)

func ConnectionString() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	host := os.Getenv("BLUEPRINT_DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("BLUEPRINT_DB_PORT")
	if port == "" {
		port = "5432"
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(os.Getenv("BLUEPRINT_DB_USERNAME"), os.Getenv("BLUEPRINT_DB_PASSWORD")), Host: net.JoinHostPort(host, port), Path: "/" + os.Getenv("BLUEPRINT_DB_DATABASE")}
	q := u.Query()
	mode := os.Getenv("DB_SSLMODE")
	if mode == "" {
		mode = "disable"
	}
	q.Set("sslmode", mode)
	if schema := os.Getenv("BLUEPRINT_DB_SCHEMA"); schema != "" {
		q.Set("search_path", schema)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
