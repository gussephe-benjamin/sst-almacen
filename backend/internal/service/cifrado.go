package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// ClavePasswords deriva la clave AES de las contraseñas asignadas a partir del secreto del servidor.
func ClavePasswords(secreto string) []byte {
	k := sha256.Sum256([]byte("sst-almacen/password-asignada/" + secreto))
	return k[:]
}

func (s *Svc) cifrar(pw string) (string, error) {
	gcm, err := s.gcm()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(pw), nil)), nil
}

func (s *Svc) descifrar(v string) (string, error) {
	gcm, err := s.gcm()
	if err != nil {
		return "", err
	}
	b, err := base64.StdEncoding.DecodeString(v)
	if err != nil || len(b) < gcm.NonceSize() {
		return "", errors.New("contraseña cifrada inválida")
	}
	pw, err := gcm.Open(nil, b[:gcm.NonceSize()], b[gcm.NonceSize():], nil)
	return string(pw), err
}

func (s *Svc) gcm() (cipher.AEAD, error) {
	if len(s.PwKey) != 32 {
		return nil, errors.New("falta la clave de contraseñas")
	}
	block, err := aes.NewCipher(s.PwKey)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

var errSinPassword = Err(http.StatusNotFound, "sin_password_asignada",
	"No hay una contraseña asignada para ver: la persona la eligió al registrarse o ya la cambió. Puedes restablecerla para entregarle una nueva.")

// guardarAsignada guarda cifrada la contraseña que el administrador entrega a la persona.
func (s *Svc) guardarAsignada(ctx context.Context, tx pgx.Tx, userID int, pw string) error {
	c, err := s.cifrar(pw)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE usuarios SET password_asignada=$2 WHERE id=$1`, userID, c)
	return err
}

// VerPassword devuelve la contraseña asignada por el administrador mientras la persona no la cambie.
func (s *Svc) VerPassword(ctx context.Context, a *Actor, userID int) (string, error) {
	var pw string
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var nombre string
		var cifrada *string
		var puestoID *int
		var rol *string
		if err := tx.QueryRow(ctx, `SELECT u.nombres, u.password_asignada, p.id, p.rol::text FROM usuarios u
			LEFT JOIN asignaciones_puesto ap ON ap.usuario_id = u.id AND ap.hasta IS NULL LEFT JOIN puestos p ON p.id = ap.puesto_id
			WHERE u.id=$1`, userID).Scan(&nombre, &cifrada, &puestoID, &rol); err != nil {
			return notFound(err)
		}
		if rol != nil && *rol == "admin" && userID != a.UserID {
			return Err(403, "sin_permiso", "Las cuentas de administrador se gestionan desde el archivo .env.")
		}
		if cifrada == nil {
			return errSinPassword
		}
		v, err := s.descifrar(*cifrada)
		if err != nil {
			return errSinPassword // p. ej. cambió el JWT_SECRET
		}
		pw = v
		if puestoID == nil {
			return nil
		}
		return Hist(ctx, tx, a, "puesto", *puestoID, "Vio la contraseña asignada de "+nombre)
	})
	return pw, err
}
