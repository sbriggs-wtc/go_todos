package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

const createTodosTableSQL = `
CREATE TABLE IF NOT EXISTS todos (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    completed BOOLEAN DEFAULT false
)
`

func runMigration(db *sql.DB) error {
	_, err := db.Exec(createTodosTableSQL)
	if err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}
	return nil
}

func setupHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	err = runMigration(db)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to run migration: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Migration successful! The 'todos' table is created.")
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello, World!")
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/setup", setupHandler)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
