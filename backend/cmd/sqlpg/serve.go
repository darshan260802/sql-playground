package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/darshan260802/sql-playground/backend/internal/api"
	"github.com/darshan260802/sql-playground/backend/internal/config"
	"github.com/darshan260802/sql-playground/backend/internal/lessons"
	"github.com/darshan260802/sql-playground/backend/internal/provision"
	"github.com/darshan260802/sql-playground/backend/internal/store"
)

func serve(ctx context.Context, cfg config.Config) {
	db := adminDB(cfg)
	st, err := store.Open(ctx, db, cfg.AppSchema)
	if err != nil {
		log.Fatal(err)
	}
	cat, err := lessons.Load()
	if err != nil {
		log.Fatalf("lessons: %v", err)
	}
	prov := provision.New(cfg, db)
	if err := prov.EnsureCore(ctx); err != nil {
		log.Fatalf("core dataset: %v", err)
	}
	if err := prov.EnsureReader(ctx); err != nil {
		log.Fatal(err)
	}
	srv := api.New(cfg, st, prov, cat)

	users, err := st.ListUsers(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, u := range users {
		if err := prov.EnsureUserAccount(ctx, u.ID); err != nil {
			log.Fatal(err)
		}
		var exists int
		_ = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", prov.Schema(u.ID)).Scan(&exists)
		switch {
		case u.DBStatus == "provisioning" || u.DBStatus == "resetting":
			log.Printf("resuming database copy for user %d", u.ID)
			srv.ProvisionAsync(u.ID, u.DBStatus)
		case u.DBStatus == "ready" && exists == 0:
			log.Printf("database for user %d is missing; recreating it", u.ID)
			srv.ProvisionAsync(u.ID, "provisioning")
		}
	}
	log.Printf("%d accounts, %d lessons, %d exercises", len(users), len(cat.Lessons), cat.ExerciseCount())

	go func() {
		for range time.Tick(time.Hour) {
			if err := st.PurgeExpiredSessions(context.Background()); err != nil {
				log.Printf("purge sessions: %v", err)
			}
		}
	}()

	hs := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	go func() {
		log.Printf("listening on %s", cfg.Listen)
		if err := hs.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Print("shutting down")
	sctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	_ = hs.Shutdown(sctx)
}

func admin(ctx context.Context, cfg config.Config, args []string) {
	if len(args) == 0 {
		usage()
	}
	db := adminDB(cfg)
	defer db.Close()
	st, err := store.Open(ctx, db, cfg.AppSchema)
	if err != nil {
		log.Fatal(err)
	}
	prov := provision.New(cfg, db)
	needEmail := func() *store.User {
		if len(args) < 2 {
			usage()
		}
		u, err := st.UserByEmail(ctx, args[1])
		if err != nil {
			log.Fatalf("no account with email %s", args[1])
		}
		return u
	}
	switch args[0] {
	case "list-users":
		users, err := st.ListUsers(ctx)
		if err != nil {
			log.Fatal(err)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tEMAIL\tNAME\tDATABASE\tSTORAGE\tCREATED")
		for _, u := range users {
			used, _ := prov.Usage(ctx, u.ID)
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%.0f MB\t%s\n", u.ID, u.Email, u.Name, u.DBStatus, float64(used)/(1<<20), u.CreatedAt.Format("2006-01-02"))
		}
		tw.Flush()
		fmt.Printf("%d of %d accounts used\n", len(users), cfg.MaxUsers)
	case "reset-password":
		u := needEmail()
		pw := store.RandomToken(9)
		hash, err := bcryptHash(pw)
		if err != nil {
			log.Fatal(err)
		}
		if err := st.SetPassword(ctx, u.ID, hash); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("New password for %s: %s\n(they can change it from the account menu after signing in)\n", u.Email, pw)
	case "delete-user":
		u := needEmail()
		if err := prov.DropUser(ctx, u.ID); err != nil {
			log.Fatal(err)
		}
		if err := st.DeleteUser(ctx, u.ID); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Deleted %s and their database.\n", u.Email)
	case "reset-db":
		u := needEmail()
		_ = st.SetDBStatus(ctx, u.ID, "resetting", "")
		start := time.Now()
		if err := prov.CopyCore(ctx, u.ID); err != nil {
			_ = st.SetDBStatus(ctx, u.ID, "error", err.Error())
			log.Fatal(err)
		}
		_ = st.SetDBStatus(ctx, u.ID, "ready", "")
		fmt.Printf("Restored the original data for %s in %s.\n", u.Email, time.Since(start).Round(time.Millisecond))
	default:
		usage()
	}
}
