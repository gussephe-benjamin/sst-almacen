package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Actor es la persona autenticada y el puesto que ocupa.
type Actor struct {
	UserID       int    `json:"usuario_id"`
	DNI          string `json:"dni"`
	Nombre       string `json:"nombre"`
	PuestoID     int    `json:"puesto_id"`
	PuestoNombre string `json:"puesto"`
	Rol          string `json:"rol"`
	AreaID       int    `json:"area_id"`
}

// AppError es un error listo para mostrarse al usuario.
type AppError struct {
	Status  int
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }

func Err(status int, code, msg string) *AppError { return &AppError{status, code, msg} }
func Bad(code, msg string) *AppError             { return Err(http.StatusBadRequest, code, msg) }
func Conflict(code, msg string) *AppError        { return Err(http.StatusConflict, code, msg) }

var (
	ErrNotFound  = Err(http.StatusNotFound, "no_encontrado", "No se encontró el registro.")
	ErrForbidden = Err(http.StatusForbidden, "sin_permiso", "No tienes permiso para esta acción.")
)

type Q interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Svc struct {
	DB    *pgxpool.Pool
	PwKey []byte // clave AES de las contraseñas asignadas por el administrador
}

func (s *Svc) Tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, s.DB, fn)
}

// List devuelve las filas de la consulta como un arreglo JSON generado por Postgres.
func List(ctx context.Context, q Q, sql string, args ...any) (json.RawMessage, error) {
	var out json.RawMessage
	err := q.QueryRow(ctx, "SELECT coalesce(json_agg(t), '[]'::json) FROM ("+sql+") t", args...).Scan(&out)
	return out, err
}

// One devuelve una fila como objeto JSON, o ErrNotFound.
func One(ctx context.Context, q Q, sql string, args ...any) (json.RawMessage, error) {
	var out json.RawMessage
	err := q.QueryRow(ctx, "SELECT row_to_json(t) FROM ("+sql+") t", args...).Scan(&out)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return out, err
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// Hist registra una acción en el historial dentro de la transacción.
func Hist(ctx context.Context, tx Q, a *Actor, entidad string, id int, accion string) error {
	_, err := tx.Exec(ctx, `INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id) VALUES ($1,$2,$3,$4,$5)`,
		entidad, id, accion, a.UserID, a.PuestoID)
	return err
}

// Notify crea un aviso para un rol, o para un puesto concreto si puestoID != 0.
func Notify(ctx context.Context, tx Q, rol string, puestoID int, texto, refTipo string, refID int) error {
	var p any
	if puestoID != 0 {
		p = puestoID
	}
	_, err := tx.Exec(ctx, `INSERT INTO notificaciones (para_rol, para_puesto_id, texto, ref_tipo, ref_id) VALUES ($1,$2,$3,$4,$5)`,
		rol, p, texto, refTipo, refID)
	return err
}

func Code(prefix string, id int) string { return fmt.Sprintf("%s-%03d", prefix, id) }

const histSQL = `SELECT h.created_at AS fecha, h.accion, u.nombres AS quien, p.nombre AS puesto
	FROM historial h JOIN usuarios u ON u.id = h.usuario_id JOIN puestos p ON p.id = h.puesto_id
	WHERE h.entidad = $1 AND h.entidad_id = $2 ORDER BY h.created_at`

func HistList(ctx context.Context, q Q, entidad string, id int) (json.RawMessage, error) {
	return List(ctx, q, histSQL, entidad, id)
}
