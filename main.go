package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"strconv"

	_ "github.com/lib/pq"
)

func runMigrations(db *sql.DB) error {
	migrations := []string{
		`
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			completed BOOLEAN DEFAULT false
		)
		`,
		`
		ALTER TABLE todos DROP COLUMN title
		`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
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

	err = runMigrations(db)
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
	// Parse the form data (assumes application/x-www-form-urlencoded or application/json)
	err := r.ParseForm()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %s", err), http.StatusInternalServerError)
		return
	}

	description := r.FormValue("description")
	completed := r.FormValue("completed") == "true" // Assuming the value is submitted as "true" or "false"

	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO todos (description, completed) VALUES ($1, $2) RETURNING id")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to prepare statement: %s", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	var id int64
	err = stmt.QueryRow(description, completed).Scan(&id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to insert record: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Inserted record with ID: %d", id)
}

type Todo struct {
	ID          int    `json:"id"`
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

	rows, err := db.Query("SELECT id, description, completed FROM todos ORDER BY id")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to execute query: %s", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []Todo // Slice to hold the todos

	for rows.Next() {
		var todo Todo
		err := rows.Scan(&todo.ID, &todo.Description, &todo.Completed)
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
	// Extract the todo ID from the URL path
	// The "/update/" part of the path will be trimmed, and the remaining part will be the todo ID
	idStr := strings.TrimPrefix(r.URL.Path, "/update/")

	// Check if the ID is empty
	if idStr == "" {
		http.Error(w, "Todo ID is missing", http.StatusBadRequest)
		return
	}

	// Convert the todo ID string to an integer
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Todo ID", http.StatusBadRequest)
		return
	}

	// Parse the form data
	err = r.ParseMultipartForm(10 << 20) // Set the maximum memory to 10 MB (adjust according to your needs)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %s", err), http.StatusInternalServerError)
		return
	}

	description := r.FormValue("description")
	completedStr := r.FormValue("completed") // Get the "completed" value as a string

	completed, err := strconv.ParseBool(completedStr) // Convert the string to a boolean
	if err != nil {
		http.Error(w, "Invalid completed value", http.StatusBadRequest)
		return
	}

	db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	stmt, err := db.Prepare("UPDATE todos SET description = $2, completed = $3 WHERE id = $1")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to prepare statement: %s", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, description, completed)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update record: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "Updated record successfully!")
}
  

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the form data (assumes application/x-www-form-urlencoded or application/json)
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

func bulkDeleteHandler(w http.ResponseWriter, r *http.Request) {
    // Parse the request body as JSON
    var request struct {
        Ids []int `json:"ids"`
    }
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        fmt.Println("Error decoding JSON:", err)
        http.Error(w, fmt.Sprintf("failed to parse request body: %s", err), http.StatusBadRequest)
        return
    }

    // Print the request body for debugging purposes
    fmt.Println("Request Body:", r.Body)


    db, err := sql.Open("postgres", "postgres://postgres:postgres@db:5432/mydb?sslmode=disable")
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to connect to the database: %s", err), http.StatusInternalServerError)
        return
    }
    defer db.Close()

    tx, err := db.Begin()
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to start transaction: %s", err), http.StatusInternalServerError)
        return
    }
    defer tx.Rollback() // Rollback the transaction if it is not committed

    stmt, err := tx.Prepare("DELETE FROM todos WHERE id = $1")
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to prepare statement: %s", err), http.StatusInternalServerError)
        return
    }
    defer stmt.Close()

    var deletedIds []int
    for _, id := range request.Ids {
        _, err = stmt.Exec(id)
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to delete record with ID %d: %s", id, err), http.StatusInternalServerError)
            return
        }
        deletedIds = append(deletedIds, id)
    }

    // Commit the transaction
    if err := tx.Commit(); err != nil {
        http.Error(w, fmt.Sprintf("failed to commit transaction: %s", err), http.StatusInternalServerError)
        return
    }

    // Respond with the list of deleted IDs
    response := struct {
        Success    bool  `json:"success"`
        Message    string `json:"message"`
        DeletedIds []int  `json:"deletedIds"`
    }{
        Success:    true,
        Message:    "Todos deleted successfully.",
        DeletedIds: deletedIds,
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        http.Error(w, fmt.Sprintf("failed to encode response as JSON: %s", err), http.StatusInternalServerError)
        return
    }
}


func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // Allow Content-Type header

		// Handle OPTIONS request
		if r.Method == http.MethodOptions {
			return
		}

		next(w, r)
	}
}

func main() {
	http.HandleFunc("/", corsMiddleware(helloHandler))
	http.HandleFunc("/setup", corsMiddleware(setupHandler))
	http.HandleFunc("/insert", corsMiddleware(insertHandler))
	http.HandleFunc("/select-all", corsMiddleware(selectAllHandler))
	http.HandleFunc("/update/", corsMiddleware(updateHandler))
	http.HandleFunc("/delete", corsMiddleware(deleteHandler))
	http.HandleFunc("/bulk-delete", corsMiddleware(bulkDeleteHandler))


	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
