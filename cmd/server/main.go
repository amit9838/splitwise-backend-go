package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/amit9838/splitwise-backend-go/internal/category"
	_ "modernc.org/sqlite"
)

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	// db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, err
	}
	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Splitwise Server"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var data = make(map[string]string)
	data["status"] = "healthy"
	json.NewEncoder(w).Encode(data)
}

// func setupRouter() http.Handler {
// 	category.NewHandler()
// }

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./splitwise.db"
	}
	db, err := openDatabase(dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	catStore := category.NewDBStore(db)
	if err := catStore.InitSchema(); err != nil {
		log.Fatalf("failed to initialize category schema: %v", err)
	}

	// handler := setupRouter()

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("GET /health", healthHandler)

	categoryHandler := category.NewHandler(category.NewService(catStore))
	// category
	mux.HandleFunc("POST /categories", categoryHandler.Create)
	mux.HandleFunc("GET /categories/{group_id}", categoryHandler.List)
	mux.HandleFunc("GET /categories/{group_id}/{id}", categoryHandler.Get)
	mux.HandleFunc("PUT /categories/{group_id}/{id}", categoryHandler.Update)
	mux.HandleFunc("DELETE /categories/{group_id}/{id}", categoryHandler.Delete)

	port := "8080"
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
