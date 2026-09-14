package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/amit9838/splitwise-backend-go/internal/category"
	"github.com/amit9838/splitwise-backend-go/internal/expense"
	"github.com/amit9838/splitwise-backend-go/internal/group"
	_ "modernc.org/sqlite"
)

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=1") // enable fk constraint
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
	if err := catStore.CreateIndexes(); err != nil {
		log.Fatalf("failed to create category indexes: %v", err)
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

	// --------------- grp ------------
	grpStore := group.NewDBStore(db)
	if err := grpStore.InitSchema(); err != nil {
		log.Fatalf("failed to initialize group schema: %v", err)
	}
	if err := grpStore.CreateIndexes(); err != nil {
		log.Fatalf("failed to create group indexes: %v", err)
	}
	groupHandler := group.NewHandler(group.NewService(grpStore))

	// group
	mux.HandleFunc("POST /groups", groupHandler.Create)
	mux.HandleFunc("GET /groups", groupHandler.List)
	mux.HandleFunc("GET /groups/{id}", groupHandler.Get)
	mux.HandleFunc("PUT /groups/{id}", groupHandler.Update)
	mux.HandleFunc("DELETE /groups/{id}", groupHandler.Delete)

	// Expense
	expStore := expense.NewDBStore(db)
	if err := expStore.InitSchema(); err != nil {
		log.Fatalf("failed to initialize expense schema: %v", err)
	}
	if err := expStore.CreateIndexes(); err != nil {
		log.Fatalf("failed to create expense indexes: %v", err)
	}
	expenseHandler := expense.NewHandler(expense.NewService(expStore))
	// expense
	mux.HandleFunc("POST /expenses", expenseHandler.Create)
	mux.HandleFunc("GET /expenses/group/{group_id}", expenseHandler.List)
	mux.HandleFunc("GET /expenses/{id}", expenseHandler.Get)
	mux.HandleFunc("PUT /expenses/{id}", expenseHandler.Update)
	mux.HandleFunc("DELETE /expenses/{id}", expenseHandler.Delete)

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
