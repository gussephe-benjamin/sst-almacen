package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var reDNI = regexp.MustCompile(`^\d{8}$`)

type RegisterIn struct {
	DNI      string `json:"dni"`
	Nombres  string `json:"nombres"`
	Correo   string `json:"correo"`
	Celular  string `json:"celular"`
	AreaID   int    `json:"area_id"`
	Password string `json:"password"`
}

func (s *Svc) Register(ctx context.Context, in RegisterIn) error {
	if !reDNI.MatchString(in.DNI) {
		return Bad("dni_invalido", "El DNI debe tener 8 dígitos.")
	}
	if strings.TrimSpace(in.Nombres) == "" || in.Correo == "" || in.Celular == "" {
		return Bad("datos_incompletos", "Completa nombres, correo y celular.")
	}
	if len(in.Password) < 6 {
		return Bad("password_corta", "La contraseña debe tener al menos 6 caracteres.")
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	var area any
	if in.AreaID != 0 {
		area = in.AreaID
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var exists bool
		tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM usuarios WHERE dni=$1)`, in.DNI).Scan(&exists)
		if exists {
			return Conflict("dni_registrado", "Ese DNI ya está registrado.")
		}
		if _, err := tx.Exec(ctx, `INSERT INTO usuarios (dni, nombres, correo, celular, password_hash, area_id) VALUES ($1,$2,$3,$4,$5,$6)`,
			in.DNI, strings.TrimSpace(in.Nombres), in.Correo, in.Celular, string(hash), area); err != nil {
			return err
		}
		return Notify(ctx, tx, "sst", 0, in.Nombres+" se registró con DNI y espera asignación de puesto.", "puestos", 0)
	})
}

// Login valida DNI y contraseña y devuelve el id del usuario.
func (s *Svc) Login(ctx context.Context, dni, password string) (int, error) {
	if !reDNI.MatchString(dni) {
		return 0, Bad("dni_invalido", "Ingresa un DNI de 8 dígitos.")
	}
	var id int
	var hash, estado string
	err := s.DB.QueryRow(ctx, `SELECT id, password_hash, estado FROM usuarios WHERE dni=$1`, dni).Scan(&id, &hash, &estado)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, Err(http.StatusUnauthorized, "dni_no_encontrado", "No encontramos ese DNI. Revisa el número o regístrate.")
	}
	if err != nil {
		return 0, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return 0, Err(http.StatusUnauthorized, "credenciales", "La contraseña no es correcta.")
	}
	if estado == "pendiente" {
		return 0, Err(http.StatusForbidden, "pendiente", "Tu registro está pendiente: SST aún no te asigna un puesto.")
	}
	if estado == "inactivo" {
		return 0, Err(http.StatusForbidden, "inactivo", "Ya no ocupas un puesto. Si es un error, comunícate con SST.")
	}
	if _, err := s.LoadActor(ctx, id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Svc) LoadActor(ctx context.Context, userID int) (*Actor, error) {
	a := &Actor{}
	err := s.DB.QueryRow(ctx, `SELECT u.id, u.dni, u.nombres, p.id, p.nombre, p.rol::text, p.area_id
		FROM usuarios u JOIN asignaciones_puesto ap ON ap.usuario_id = u.id AND ap.hasta IS NULL
		JOIN puestos p ON p.id = ap.puesto_id WHERE u.id = $1 AND u.estado = 'activo'`, userID).
		Scan(&a.UserID, &a.DNI, &a.Nombre, &a.PuestoID, &a.PuestoNombre, &a.Rol, &a.AreaID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, Err(http.StatusForbidden, "pendiente", "Tu registro está pendiente: SST aún no te asigna un puesto.")
	}
	return a, err
}
