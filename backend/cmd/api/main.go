package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sst-almacen/internal/config"
	"sst-almacen/internal/httpapi"
	"sst-almacen/internal/service"
	"sst-almacen/internal/storage"
	"sst-almacen/migrations"
)

const uso = `Uso: api [comando]
  (sin comando)  inicia la API
  migrate        aplica las migraciones
  reset --si     BORRA todos los datos y archivos y deja la base lista para uso real
                 (catálogos + cuentas del .env)
  demo  --si     BORRA todo y carga datos de demostración (solo desarrollo)`

func main() {
	cfg := config.Load()
	cmd := "serve"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	ctx := context.Background()

	switch cmd {
	case "serve", "migrate", "reset", "demo":
	default:
		log.Fatal(uso)
	}
	if (cmd == "reset" || cmd == "demo") && (len(os.Args) < 3 || os.Args[2] != "--si") {
		log.Fatalf("%q borra todos los datos. Para confirmar ejecuta: api %s --si", cmd, cmd)
	}
	if cmd == "demo" && cfg.AppEnv == "production" {
		log.Fatal("no se cargan datos de demostración con APP_ENV=production")
	}

	// La API aplica las migraciones pendientes al iniciar (en la nube no hay un paso aparte).
	if cmd == "migrate" || cmd == "serve" {
		if err := migrar(cfg.DatabaseURL); err != nil {
			log.Fatal(err)
		}
		log.Println("migraciones aplicadas")
	}
	if cmd == "migrate" {
		return
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	files := &storage.Store{Dir: cfg.UploadDir}
	svc := &service.Svc{DB: pool, PwKey: service.ClavePasswords(cfg.JWTSecret)}

	switch cmd {
	case "reset":
		if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return resetDB(ctx, tx, files) }); err != nil {
			log.Fatal(err)
		}
		log.Println("base reiniciada: solo catálogos")
		svc.EnsureAccounts(ctx, cfg.Accounts)
		return
	case "demo":
		if err := seed(ctx, pool, files); err != nil {
			log.Fatal(err)
		}
		log.Println("datos de demostración cargados")
		svc.EnsureAccounts(ctx, cfg.Accounts)
		return
	}

	if cfg.AppEnv == "production" {
		if p := cfg.Problems(); len(p) > 0 {
			for _, x := range p {
				log.Println("configuración insegura:", x)
			}
			log.Fatal("corrige el .env antes de iniciar en producción")
		}
	}
	svc.EnsureAccounts(ctx, cfg.Accounts)
	api := &httpapi.API{S: svc, Files: files, Secret: cfg.JWTSecret, CookieSecure: cfg.CookieSecure, StaticDir: cfg.StaticDir, TrustProxy: cfg.TrustProxy}
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: api.Router(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second}
	log.Printf("API en :%s (%s)", cfg.Port, cfg.AppEnv)
	log.Fatal(srv.ListenAndServe())
}

// migrar aplica las migraciones embebidas. Acepta URLs postgres:// y postgresql:// (Railway).
func migrar(dbURL string) error {
	_, rest, ok := strings.Cut(dbURL, "://")
	if !ok {
		return errors.New("DATABASE_URL no válida")
	}
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+rest)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
