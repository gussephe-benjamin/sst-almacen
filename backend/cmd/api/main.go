package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"sst-almacen/internal/config"
	"sst-almacen/internal/httpapi"
	"sst-almacen/internal/service"
	"sst-almacen/internal/storage"
	"sst-almacen/migrations"
)

func main() {
	cfg := config.Load()
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	ctx := context.Background()

	if cmd == "migrate" {
		src, _ := iofs.New(migrations.FS, ".")
		m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5"+cfg.DatabaseURL[len("postgres"):])
		if err != nil {
			log.Fatal(err)
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal(err)
		}
		log.Println("migraciones aplicadas")
		return
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	files := &storage.Store{Dir: cfg.UploadDir}

	if cmd == "seed" {
		if err := seed(ctx, pool, files); err != nil {
			log.Fatal(err)
		}
		log.Println("datos de demostración cargados")
		return
	}

	api := &httpapi.API{S: &service.Svc{DB: pool}, Files: files, Secret: cfg.JWTSecret}
	log.Printf("API en :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, api.Router()))
}
