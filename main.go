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

func insertTodo(db *sql.DB, title, description string, completed bool) (int64, error) {
	stmt, err := db.Prepare("INSERT INTO todos (title, description, completed) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	var id int64
	err = stmt.QueryRow(title, description, completed).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to insert record: %w", err)
	}

	return id, nil
}

func insertHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse form values (assuming you are submitting form data for title, description, and completed)
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %s", err), http.StatusInternalServerError)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	completed := r.FormValue("completed") == "true" // Assuming the value is submitted as "true" or "false"

	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	id, err := insertTodo(db, title, description, completed)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to insert record: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Inserted record with ID: %d", id)
}

func main() {
	http.HandleFunc("/", helloHandler)
	http.HandleFunc("/setup", setupHandler)
	http.HandleFunc("/insert", insertHandler) // New endpoint for inserting records

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
