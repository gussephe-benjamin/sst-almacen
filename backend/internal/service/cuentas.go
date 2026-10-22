package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"sst-almacen/internal/config"
)

// Área por defecto de cada rol cuando la cuenta del .env no indica una.
var areaPorRol = map[string]string{
	"admin": "Dirección", "sst": "SST", "trabajador": "Almacén A", "mantenimiento": "Mantenimiento", "administracion": "Administración",
}

var puestoPorRol = map[string]string{
	"admin": "Administrador del sistema", "sst": "Especialista SST", "trabajador": "Operario de almacén",
	"mantenimiento": "Técnico de mantenimiento", "administracion": "Asistente de administración",
}

// Rol que el admin puede crear desde la interfaz. Las cuentas admin solo se definen en el .env.
var rolesGestionables = map[string]bool{"trabajador": true, "sst": true, "mantenimiento": true, "administracion": true}

func hashPw(pw string) string {
	h, _ := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h)
}

// EnsureAccounts crea las cuentas definidas en el .env que aún no existen.
// Si el DNI ya existe no se modifica (la contraseña puede haberse cambiado desde la app).
func (s *Svc) EnsureAccounts(ctx context.Context, accounts map[string][]config.Account) {
	for _, rol := range config.Roles {
		for _, acc := range accounts[rol] {
			created, err := s.ensureAccount(ctx, rol, acc)
			switch {
			case err != nil:
				log.Printf("cuenta %s (%s): %v", acc.DNI, rol, err)
			case created:
				log.Printf("cuenta creada desde .env: %s (%s)", acc.DNI, rol)
			}
		}
	}
}

func (s *Svc) ensureAccount(ctx context.Context, rol string, acc config.Account) (bool, error) {
	if !reDNI.MatchString(acc.DNI) || acc.Password == "" || acc.Nombres == "" {
		return false, errors.New("formato inválido: se espera DNI|contraseña|Nombres|Puesto|Área|Correo|Celular")
	}
	created := false
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var exists bool
		tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM usuarios WHERE dni=$1)`, acc.DNI).Scan(&exists)
		if exists {
			return nil
		}
		area := acc.Area
		if area == "" {
			area = areaPorRol[rol]
		}
		var areaID int
		if err := tx.QueryRow(ctx, `SELECT id FROM areas WHERE nombre=$1`, area).Scan(&areaID); err != nil {
			return fmt.Errorf("no existe el área %q", area)
		}
		puesto := acc.Puesto
		if puesto == "" {
			puesto = puestoPorRol[rol]
		}
		var puestoID int
		var ocupado bool
		err := tx.QueryRow(ctx, `SELECT p.id, EXISTS(SELECT 1 FROM asignaciones_puesto ap WHERE ap.puesto_id = p.id AND ap.hasta IS NULL)
			FROM puestos p WHERE p.nombre=$1 AND p.rol=$2 ORDER BY p.id LIMIT 1`, puesto, rol).Scan(&puestoID, &ocupado)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `INSERT INTO puestos (nombre, rol, area_id) VALUES ($1,$2,$3) RETURNING id`, puesto, rol, areaID).Scan(&puestoID)
		}
		if err != nil {
			return err
		}
		if ocupado {
			return fmt.Errorf("el puesto %q ya está ocupado por otra persona", puesto)
		}
		var userID int
		if err := tx.QueryRow(ctx, `INSERT INTO usuarios (dni, nombres, correo, celular, password_hash, area_id, estado)
			VALUES ($1,$2,$3,$4,$5,$6,'activo') RETURNING id`, acc.DNI, acc.Nombres, acc.Correo, acc.Celular, hashPw(acc.Password), areaID).Scan(&userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO asignaciones_puesto (puesto_id, usuario_id) VALUES ($1,$2)`, puestoID, userID); err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

type CuentaIn struct {
	DNI          string `json:"dni"`
	Nombres      string `json:"nombres"`
	Correo       string `json:"correo"`
	Celular      string `json:"celular"`
	Password     string `json:"password"`
	Rol          string `json:"rol"`
	PuestoID     int    `json:"puesto_id"`
	PuestoNombre string `json:"puesto_nombre"`
	AreaID       int    `json:"area_id"`
}

// CrearCuenta crea a la persona y la asigna a un puesto existente o a uno nuevo.
// Si el puesto existente está ocupado, la persona actual sale del puesto.
func (s *Svc) CrearCuenta(ctx context.Context, a *Actor, in CuentaIn) (int, error) {
	in.Nombres, in.PuestoNombre = strings.TrimSpace(in.Nombres), strings.TrimSpace(in.PuestoNombre)
	switch {
	case !reDNI.MatchString(in.DNI):
		return 0, Bad("dni_invalido", "El DNI debe tener 8 dígitos.")
	case in.Nombres == "":
		return 0, Bad("datos_incompletos", "Ingresa los nombres y apellidos.")
	case len(in.Password) < 6:
		return 0, Bad("password_corta", "La contraseña debe tener al menos 6 caracteres.")
	case in.PuestoID == 0 && !rolesGestionables[in.Rol]:
		return 0, Bad("rol_invalido", "Elige el rol: Trabajador, SST, Mantenimiento o Administración.")
	case in.PuestoID == 0 && (in.PuestoNombre == "" || in.AreaID == 0):
		return 0, Bad("datos_incompletos", "Indica el nombre del puesto y su área.")
	}
	var userID int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var exists bool
		tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM usuarios WHERE dni=$1)`, in.DNI).Scan(&exists)
		if exists {
			return Conflict("dni_registrado", "Ese DNI ya tiene una cuenta. Si no tiene puesto, asígnalo desde “Personas sin puesto”.")
		}
		puestoID, rol := in.PuestoID, in.Rol
		var puesto string
		if puestoID == 0 {
			if err := tx.QueryRow(ctx, `INSERT INTO puestos (nombre, rol, area_id) VALUES ($1,$2,$3) RETURNING id, nombre`,
				in.PuestoNombre, in.Rol, in.AreaID).Scan(&puestoID, &puesto); err != nil {
				return err
			}
			if err := Hist(ctx, tx, a, "puesto", puestoID, "Creó el puesto "+puesto); err != nil {
				return err
			}
		} else if err := tx.QueryRow(ctx, `SELECT nombre, rol::text FROM puestos WHERE id=$1 FOR UPDATE`, puestoID).Scan(&puesto, &rol); err != nil {
			return notFound(err)
		}
		if !rolesGestionables[rol] {
			return ErrForbidden
		}
		var areaID int
		tx.QueryRow(ctx, `SELECT area_id FROM puestos WHERE id=$1`, puestoID).Scan(&areaID)
		if err := tx.QueryRow(ctx, `INSERT INTO usuarios (dni, nombres, correo, celular, password_hash, area_id, estado)
			VALUES ($1,$2,$3,$4,$5,$6,'activo') RETURNING id`, in.DNI, in.Nombres, in.Correo, in.Celular, hashPw(in.Password), areaID).Scan(&userID); err != nil {
			return err
		}
		if err := liberar(ctx, tx, puestoID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO asignaciones_puesto (puesto_id, usuario_id, asignado_por) VALUES ($1,$2,$3)`, puestoID, userID, a.UserID); err != nil {
			return err
		}
		return Hist(ctx, tx, a, "puesto", puestoID, fmt.Sprintf("Creó la cuenta de %s (DNI %s) en el puesto %s", in.Nombres, in.DNI, puesto))
	})
	return userID, err
}

// liberar cierra la asignación activa del puesto y deja inactiva a esa persona.
func liberar(ctx context.Context, tx pgx.Tx, puestoID int) error {
	var anterior *int
	tx.QueryRow(ctx, `UPDATE asignaciones_puesto SET hasta=now() WHERE puesto_id=$1 AND hasta IS NULL RETURNING usuario_id`, puestoID).Scan(&anterior)
	if anterior == nil {
		return nil
	}
	_, err := tx.Exec(ctx, `UPDATE usuarios SET estado='inactivo' WHERE id=$1`, *anterior)
	return err
}

func (s *Svc) LiberarPuesto(ctx context.Context, a *Actor, puestoID int) error {
	if puestoID == a.PuestoID {
		return Conflict("propio_puesto", "No puedes liberar tu propio puesto.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var puesto, rol string
		var persona *string
		if err := tx.QueryRow(ctx, `SELECT p.nombre, p.rol::text, u.nombres FROM puestos p
			LEFT JOIN asignaciones_puesto ap ON ap.puesto_id = p.id AND ap.hasta IS NULL LEFT JOIN usuarios u ON u.id = ap.usuario_id
			WHERE p.id=$1`, puestoID).Scan(&puesto, &rol, &persona); err != nil {
			return notFound(err)
		}
		if !rolesGestionables[rol] {
			return ErrForbidden
		}
		if persona == nil {
			return Conflict("puesto_vacante", "El puesto ya está vacante.")
		}
		if err := liberar(ctx, tx, puestoID); err != nil {
			return err
		}
		return Hist(ctx, tx, a, "puesto", puestoID, fmt.Sprintf("Retiró a %s del puesto %s", *persona, puesto))
	})
}

func (s *Svc) ResetPassword(ctx context.Context, a *Actor, userID int, pw string) error {
	if len(pw) < 6 {
		return Bad("password_corta", "La contraseña debe tener al menos 6 caracteres.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var nombre string
		var puestoID *int
		var rol *string
		if err := tx.QueryRow(ctx, `SELECT u.nombres, p.id, p.rol::text FROM usuarios u
			LEFT JOIN asignaciones_puesto ap ON ap.usuario_id = u.id AND ap.hasta IS NULL LEFT JOIN puestos p ON p.id = ap.puesto_id
			WHERE u.id=$1`, userID).Scan(&nombre, &puestoID, &rol); err != nil {
			return notFound(err)
		}
		if rol != nil && *rol == "admin" && userID != a.UserID {
			return Err(403, "sin_permiso", "Las cuentas de administrador se gestionan desde el archivo .env.")
		}
		if _, err := tx.Exec(ctx, `UPDATE usuarios SET password_hash=$2 WHERE id=$1`, userID, hashPw(pw)); err != nil {
			return err
		}
		if puestoID == nil {
			return nil
		}
		return Hist(ctx, tx, a, "puesto", *puestoID, "Restableció la contraseña de "+nombre)
	})
}
