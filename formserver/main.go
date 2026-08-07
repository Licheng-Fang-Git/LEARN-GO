// Command formserver is a small, dependency-light web server that hosts forms
// generated from JSON templates, collects submissions over HTTP, stores them in
// SQLite via CRUD operations, and reports the outcome back to the client.
//
// Everything is configurable by flag so you can point it at a different port,
// database file, or set of form definitions without touching the code.
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"formserver/internal/forms"
	"formserver/internal/store"
	"formserver/internal/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "formserver.db", "path to the SQLite database file")
	formsDir := flag.String("forms", "forms", "directory of form-definition JSON files")
	tmplDir := flag.String("templates", "templates", "directory of HTML templates")
	staticDir := flag.String("static", "static", "directory of static assets")
	flag.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmsgprefix)

	// 1. Load the form templates.
	registry, err := forms.LoadDir(*formsDir)
	if err != nil {
		logger.Fatalf("loading forms: %v", err)
	}
	logger.Printf("loaded %d form(s) from %q", registry.Len(), *formsDir)

	// 2. Open and migrate the database.
	st, err := store.Open(*dbPath)
	if err != nil {
		logger.Fatalf("opening database: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(context.Background()); err != nil {
		logger.Fatalf("migrating database: %v", err)
	}

	// 3. Build the HTTP application.
	app, err := web.New(registry, st, *tmplDir, *staticDir, logger)
	if err != nil {
		logger.Fatalf("initialising web layer: %v", err)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// 4. Serve, with graceful shutdown on Ctrl-C / SIGTERM.
	go func() {
		logger.Printf("listening on http://localhost%s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("shutdown error: %v", err)
	}
	logger.Println("bye")
}
