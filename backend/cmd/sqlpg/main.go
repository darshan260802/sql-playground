// Command sqlpg runs the SQL playground API server and its admin tasks.
//
//	sqlpg serve                          start the HTTP API
//	sqlpg seed                           (re)generate the core dataset
//	sqlpg admin list-users               show accounts and storage use
//	sqlpg admin reset-password <email>   set a new password and print it
//	sqlpg admin delete-user <email>      remove an account and its database
//	sqlpg admin reset-db <email>         restore a user's database to the original data
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"

	"github.com/darshan260802/sql-playground/backend/internal/config"
	"github.com/darshan260802/sql-playground/backend/internal/dataset"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	if len(os.Args) < 2 {
		usage()
	}
	cfg := config.Load()
	ctx := context.Background()
	switch os.Args[1] {
	case "serve":
		serve(ctx, cfg)
	case "seed":
		seed(ctx, cfg)
	case "admin":
		admin(ctx, cfg, os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  sqlpg serve
  sqlpg seed
  sqlpg admin list-users
  sqlpg admin reset-password <email>
  sqlpg admin delete-user <email>
  sqlpg admin reset-db <email>`)
	os.Exit(2)
}

// adminDB opens a pool using the MySQL admin account.
func adminDB(cfg config.Config) *sql.DB {
	mc := mysql.NewConfig()
	mc.User = cfg.MySQLAdminUser
	mc.Passwd = cfg.MySQLAdminPassword
	mc.Net = "tcp"
	mc.Addr = cfg.MySQLAddr
	mc.ParseTime = true
	mc.AllowNativePasswords = true
	mc.Params = map[string]string{"charset": "utf8mb4"}
	mc.Collation = "utf8mb4_0900_ai_ci"
	db, err := sql.Open("mysql", mc.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	for i := 0; ; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return db
		}
		if i == 30 {
			log.Fatalf("cannot reach MySQL at %s: %v", cfg.MySQLAddr, err)
		}
		log.Printf("waiting for MySQL: %v", err)
		time.Sleep(2 * time.Second)
	}
}

func seed(ctx context.Context, cfg config.Config) {
	db := adminDB(cfg)
	defer db.Close()
	err := dataset.Generate(ctx, db, dataset.Options{
		Schema: cfg.CoreSchema,
		Log:    func(f string, a ...any) { log.Printf(f, a...) },
	})
	if err != nil {
		log.Fatal(err)
	}
	var mb float64
	err = db.QueryRowContext(ctx, `SELECT COALESCE(SUM(FILE_SIZE),0)/1048576 FROM information_schema.INNODB_TABLESPACES WHERE NAME = ?`, dataset.TablespaceName(cfg.CoreSchema)).Scan(&mb)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("core dataset size on disk: %.1f MB", mb)
}

func bcryptHash(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(h), err
}
