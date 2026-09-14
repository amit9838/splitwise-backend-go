package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/amit9838/splitwise-backend-go/internal/apidocs"
	"github.com/amit9838/splitwise-backend-go/internal/balance"
	"github.com/amit9838/splitwise-backend-go/internal/category"
	"github.com/amit9838/splitwise-backend-go/internal/expense"
	"github.com/amit9838/splitwise-backend-go/internal/group"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/auth"
	"github.com/amit9838/splitwise-backend-go/internal/pkg/response"
	"github.com/amit9838/splitwise-backend-go/internal/settlement"
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
	mux.HandleFunc("GET /docs", apidocs.SwaggerUI)
	mux.HandleFunc("GET /docs/openapi.yaml", apidocs.Spec)
	mux.HandleFunc("GET /redoc", apidocs.ReDoc)

	requireAuth := auth.RequireAuth(authMgr)
	register := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, requireAuth(h))
	}

	// --------------- user ------------
	userStore := user.NewDBStore(db)
	mustInitSchema("user", userStore)
	userHandler := user.NewHandler(user.NewService(userStore), authMgr)

	// auth (public)
	mux.HandleFunc("POST /api/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/auth/login", userHandler.Login)
	mux.HandleFunc("POST /api/auth/refresh", userHandler.Refresh)
	// users
	mux.HandleFunc("POST /api/users", userHandler.Create)
	register("GET /api/auth/me", userHandler.Me)
	register("GET /api/users", userHandler.List)
	register("GET /api/users/{id}", userHandler.Get)
	register("PUT /api/users/{id}", userHandler.Update)
	register("DELETE /api/users/{id}", userHandler.Delete)

	// --------------- category ------------
	catStore := category.NewDBStore(db)
	mustInitSchema("category", catStore)
	categoryHandler := category.NewHandler(category.NewService(catStore))
	register("POST /api/categories", categoryHandler.Create)
	register("GET /api/categories/{group_id}", categoryHandler.List)
	register("GET /api/categories/{group_id}/{id}", categoryHandler.Get)
	register("PUT /api/categories/{group_id}/{id}", categoryHandler.Update)
	register("DELETE /api/categories/{group_id}/{id}", categoryHandler.Delete)

	// --------------- group ------------
	grpStore := group.NewDBStore(db)
	mustInitSchema("group", grpStore)
	memberStore := group.NewMemberDBStore(db)
	mustInitSchema("group member", memberStore)
	groupHandler := group.NewHandler(group.NewService(grpStore, memberStore, userStore))
	register("POST /api/groups", groupHandler.Create)
	register("GET /api/groups", groupHandler.List)
	register("GET /api/groups/{id}", groupHandler.Get)
	register("PUT /api/groups/{id}", groupHandler.Update)
	register("DELETE /api/groups/{id}", groupHandler.Delete)
	register("POST /api/groups/{group_id}/members", groupHandler.AddMember)
	register("DELETE /api/groups/{group_id}/members/{user_id}", groupHandler.RemoveMember)

	// --------------- expense ------------
	expStore := expense.NewDBStore(db)
	mustInitSchema("expense", expStore)
	expenseHandler := expense.NewHandler(expense.NewService(expStore, grpStore, memberStore))
	register("POST /api/expenses", expenseHandler.Create)
	register("GET /api/expenses/group/{group_id}", expenseHandler.List)
	register("GET /api/expenses/{id}", expenseHandler.Get)
	register("PUT /api/expenses/{id}", expenseHandler.Update)
	register("DELETE /api/expenses/{id}", expenseHandler.Delete)

	// --------------- settlement ------------
	setStore := settlement.NewDBStore(db)
	mustInitSchema("settlement", setStore)
	settlementHandler := settlement.NewHandler(settlement.NewService(setStore, grpStore, memberStore))
	register("POST /api/settlements", settlementHandler.Create)
	register("GET /api/settlements/group/{group_id}", settlementHandler.List)
	register("DELETE /api/settlements/{id}", settlementHandler.Delete)

	// Balances
	balanceHandler := balance.NewHandler(balance.NewService(expStore, setStore, grpStore, memberStore, userStore))
	// balance
	register("GET /api/balances/group/{group_id}", balanceHandler.Group)
	register("GET /api/balances/me", balanceHandler.Me)

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
