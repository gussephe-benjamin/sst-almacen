package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"sst-almacen/internal/service"
)

type ctxKey struct{}

func actor(r *http.Request) *service.Actor { return r.Context().Value(ctxKey{}).(*service.Actor) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var ae *service.AppError
	if !errors.As(err, &ae) {
		log.Printf("error interno: %v", err)
		ae = service.Err(http.StatusInternalServerError, "error_interno", "Ocurrió un error inesperado. Intenta de nuevo.")
	}
	writeJSON(w, ae.Status, map[string]any{"error": map[string]string{"code": ae.Code, "message": ae.Message}})
}

// reply escribe v o el error.
func reply(w http.ResponseWriter, v any, err error) {
	if err != nil {
		writeErr(w, err)
		return
	}
	if raw, ok := v.(json.RawMessage); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(raw)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func ok(w http.ResponseWriter, err error) { reply(w, map[string]bool{"ok": true}, err) }

func decode(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return service.Bad("json_invalido", "Los datos enviados no son válidos.")
	}
	return nil
}

func idParam(r *http.Request, name string) int {
	n, _ := strconv.Atoi(chi.URLParam(r, name))
	return n
}

const cookieName = "sst_session"

const sessionTTL = 7 * 24 * time.Hour

func (a *API) issue(w http.ResponseWriter, userID int) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID, "exp": time.Now().Add(sessionTTL).Unix(),
	}).SignedString([]byte(a.Secret))
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: tok, Path: "/", HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds())})
}

func (a *API) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			writeErr(w, service.Err(http.StatusUnauthorized, "sin_sesion", "Inicia sesión para continuar."))
			return
		}
		tok, err := jwt.Parse(c.Value, func(t *jwt.Token) (any, error) { return []byte(a.Secret), nil }, jwt.WithValidMethods([]string{"HS256"}))
		if err != nil || !tok.Valid {
			writeErr(w, service.Err(http.StatusUnauthorized, "sin_sesion", "Tu sesión expiró. Inicia sesión de nuevo."))
			return
		}
		f, _ := tok.Claims.(jwt.MapClaims)["sub"].(float64)
		act, err := a.S.LoadActor(r.Context(), int(f))
		if err != nil {
			writeErr(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, act)))
	})
}

// RequireRole restringe una ruta a los roles indicados.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rol := actor(r).Rol
			for _, x := range roles {
				if x == rol {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeErr(w, service.ErrForbidden)
		})
	}
}
