package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/amit9838/splitwise-backend-go/internal/category"
	"github.com/amit9838/splitwise-backend-go/internal/expense"
	"github.com/amit9838/splitwise-backend-go/internal/group"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/user"
	_ "modernc.org/sqlite"
)

// schemaInitializer is implemented by every DB store.
type schemaInitializer interface {
	InitSchema() error
	CreateIndexes() error
}

// mustInitSchema creates the schema and indexes for a store, aborting on error.
func mustInitSchema(name string, store schemaInitializer) {
	if err := store.InitSchema(); err != nil {
		log.Fatalf("failed to initialize %s schema: %v", name, err)
	}
	if err := store.CreateIndexes(); err != nil {
		log.Fatalf("failed to create %s indexes: %v", name, err)
	}
}

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath+"?_foreign_keys=1") // enable fk constraint
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func newAuthManager() *auth.Manager {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
		log.Println("WARNING: JWT_SECRET not set, using insecure development default")
	}
	return auth.NewManager(jwtSecret)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Splitwise Server"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func setupRouter(db *sql.DB, authMgr *auth.Manager) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("GET /health", healthHandler)

	requireAuth := auth.RequireAuth(authMgr)
	register := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, requireAuth(h))
	}

	// --------------- user ------------
	userStore := user.NewDBStore(db)
	mustInitSchema("user", userStore)
	userHandler := user.NewHandler(user.NewService(userStore), authMgr)

	// auth (public)
	mux.HandleFunc("POST /auth/register", userHandler.Register)
	mux.HandleFunc("POST /auth/login", userHandler.Login)
	mux.HandleFunc("POST /auth/refresh", userHandler.Refresh)
	// users
	mux.HandleFunc("POST /users", userHandler.Create)
	register("GET /auth/me", userHandler.Me)
	register("GET /users", userHandler.List)
	register("GET /users/{id}", userHandler.Get)
	register("PUT /users/{id}", userHandler.Update)
	register("DELETE /users/{id}", userHandler.Delete)

	// --------------- category ------------
	catStore := category.NewDBStore(db)
	mustInitSchema("category", catStore)
	categoryHandler := category.NewHandler(category.NewService(catStore))
	register("POST /categories", categoryHandler.Create)
	register("GET /categories/{group_id}", categoryHandler.List)
	register("GET /categories/{group_id}/{id}", categoryHandler.Get)
	register("PUT /categories/{group_id}/{id}", categoryHandler.Update)
	register("DELETE /categories/{group_id}/{id}", categoryHandler.Delete)

	// --------------- group ------------
	grpStore := group.NewDBStore(db)
	mustInitSchema("group", grpStore)
	memberStore := group.NewMemberDBStore(db)
	mustInitSchema("group member", memberStore)
	groupHandler := group.NewHandler(group.NewService(grpStore, memberStore, userStore))
	register("POST /groups", groupHandler.Create)
	register("GET /groups", groupHandler.List)
	register("GET /groups/{id}", groupHandler.Get)
	register("PUT /groups/{id}", groupHandler.Update)
	register("DELETE /groups/{id}", groupHandler.Delete)
	register("POST /groups/{group_id}/members", groupHandler.AddMember)
	register("DELETE /groups/{group_id}/members/{user_id}", groupHandler.RemoveMember)

	// --------------- expense ------------
	expStore := expense.NewDBStore(db)
	mustInitSchema("expense", expStore)
	expenseHandler := expense.NewHandler(expense.NewService(expStore))
	register("POST /expenses", expenseHandler.Create)
	register("GET /expenses/group/{group_id}", expenseHandler.List)
	register("GET /expenses/{id}", expenseHandler.Get)
	register("PUT /expenses/{id}", expenseHandler.Update)
	register("DELETE /expenses/{id}", expenseHandler.Delete)

	return mux
}

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

	authMgr := newAuthManager()

	port := "8080"
	server := &http.Server{
		Addr:    ":" + port,
		Handler: setupRouter(db, authMgr),
	}
	log.Printf("Server listening on http://localhost:%s\n", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
