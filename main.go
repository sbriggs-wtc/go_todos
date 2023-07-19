package main

import (
	"database/sql"
	"encoding/json"
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

func insertHandler(w http.ResponseWriter, r *http.Request) {

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

	stmt, err := db.Prepare("INSERT INTO todos (title, description, completed) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to prepare statement: %s", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var id int64
	err = stmt.QueryRow(title, description, completed).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to insert record: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Inserted record with ID: %d", id)
}

type Todo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func selectAllHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, title, description, completed FROM todos")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute query: %s", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo // Slice to hold the todos

	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Completed)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to scan row: %s", err), http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	// Encode the todos slice as JSON and write it to the response
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(todos)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode todos as JSON: %s", err), http.StatusInternalServerError)
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	// Parse form values (assuming you are submitting form data for title, description, and completed)
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %s", err), http.StatusInternalServerError)
		return
	}

	id := r.FormValue("id")
	title := r.FormValue("title")
	description := r.FormValue("description")
	completed := r.FormValue("completed") == "true" // Assuming the value is submitted as "true" or "false"

	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE todos SET title = $2, description = $3, completed = $4 WHERE id = $1")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to prepare statement: %s", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, title, description, completed)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update record: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Updated record successfully!")
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	// Parse form values (assuming you are submitting form data for the ID of the record to delete)
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %s", err), http.StatusInternalServerError)
		return
	}

	id := r.FormValue("id")

	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("DELETE FROM todos WHERE id = $1")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to prepare statement: %s", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete record: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Deleted record successfully!")
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next(w, r)
	}
}

func main() {
	http.HandleFunc("/", corsMiddleware(helloHandler))
	http.HandleFunc("/setup", corsMiddleware(setupHandler))
	http.HandleFunc("/insert", corsMiddleware(insertHandler))
	http.HandleFunc("/select-all", corsMiddleware(selectAllHandler))
	http.HandleFunc("/update", corsMiddleware(updateHandler))
	http.HandleFunc("/delete", deleteHandler)   
	
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
