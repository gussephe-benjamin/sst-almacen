package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	UploadDir   string
	Port        string
	// Cuentas iniciales por rol, leídas de ACCOUNTS_<ROL>.
	Accounts map[string][]Account
}

// Account es una cuenta definida en el .env:
// DNI|contraseña|Nombres|Puesto|Área|Correo|Celular (Puesto, Área, Correo y Celular son opcionales).
type Account struct {
	DNI, Password, Nombres, Puesto, Area, Correo, Celular string
}

var Roles = []string{"admin", "sst", "trabajador", "mantenimiento", "administracion"}

func get(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// loadDotEnv carga .env (o ../.env) sin pisar variables ya definidas en el entorno.
func loadDotEnv() {
	for _, p := range []string{".env", "../.env"} {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			k, v = strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"`)
			if _, exists := os.LookupEnv(k); !exists {
				os.Setenv(k, v)
			}
		}
		f.Close()
		return
	}
}

func parseAccounts(s string) []Account {
	var out []Account
	for _, entry := range strings.Split(s, ";") {
		f := strings.Split(strings.TrimSpace(entry), "|")
		if len(f) < 3 {
			continue
		}
		for len(f) < 7 {
			f = append(f, "")
		}
		for i := range f {
			f[i] = strings.TrimSpace(f[i])
		}
		out = append(out, Account{DNI: f[0], Password: f[1], Nombres: f[2], Puesto: f[3], Area: f[4], Correo: f[5], Celular: f[6]})
	}
	return out
}

func Load() Config {
	loadDotEnv()
	c := Config{
		DatabaseURL: get("DATABASE_URL", "postgres://sst:sst@localhost:5433/sst?sslmode=disable"),
		JWTSecret:   get("JWT_SECRET", "dev-secret-cambiar"),
		UploadDir:   get("UPLOAD_DIR", "./data/uploads"),
		Port:        get("PORT", "8081"),
		Accounts:    map[string][]Account{},
	}
	for _, r := range Roles {
		c.Accounts[r] = parseAccounts(os.Getenv("ACCOUNTS_" + strings.ToUpper(r)))
	}
	return c
}
