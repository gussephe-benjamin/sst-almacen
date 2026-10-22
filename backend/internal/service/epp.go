package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const eppSQL = `SELECT e.id, 'E-' || lpad(e.id::text, 3, '0') AS codigo, e.estado, e.motivo, e.descripcion,
	e.foto_id, e.fecha_espera, e.motivo_rechazo, e.epp_anterior_devuelto, e.formato_firmado, e.entregada_at, e.created_at,
	r.id AS recurso_id, r.nombre AS recurso, r.stock, p.id AS puesto_id, p.nombre AS puesto, u.nombres AS trabajador, u.dni
	FROM solicitudes_epp e JOIN recursos r ON r.id = e.recurso_id JOIN puestos p ON p.id = e.puesto_id
	JOIN usuarios u ON u.id = e.usuario_id`

type EppIn struct {
	RecursoID   int    `json:"recurso_id"`
	Motivo      string `json:"motivo"`
	Descripcion string `json:"descripcion"`
	FotoID      string `json:"foto_id"`
}

func (s *Svc) EppCrear(ctx context.Context, a *Actor, in EppIn) (int, error) {
	if in.RecursoID == 0 || in.Motivo == "" || strings.TrimSpace(in.Descripcion) == "" {
		return 0, Bad("datos_incompletos", "Elige el EPP, el motivo y describe el problema.")
	}
	if in.FotoID == "" {
		return 0, Bad("foto_requerida", "Adjunta una foto del EPP dañado.")
	}
	var id int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var recurso string
		if err := tx.QueryRow(ctx, `SELECT nombre FROM recursos WHERE id=$1`, in.RecursoID).Scan(&recurso); err != nil {
			return notFound(err)
		}
		if err := tx.QueryRow(ctx, `INSERT INTO solicitudes_epp (puesto_id, usuario_id, recurso_id, motivo, descripcion, foto_id)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, a.PuestoID, a.UserID, in.RecursoID, in.Motivo, in.Descripcion, in.FotoID).Scan(&id); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "solicitud_epp", id, "Envió la solicitud"); err != nil {
			return err
		}
		return Notify(ctx, tx, "sst", 0, fmt.Sprintf("%s solicitó cambio de %s (%s).", a.Nombre, recurso, Code("E", id)), "epp", id)
	})
	return id, err
}

func (s *Svc) EppMias(ctx context.Context, a *Actor) (json.RawMessage, error) {
	return List(ctx, s.DB, eppSQL+` WHERE e.puesto_id = $1 ORDER BY e.created_at DESC`, a.PuestoID)
}

func (s *Svc) EppLista(ctx context.Context, estado string) (json.RawMessage, error) {
	return List(ctx, s.DB, eppSQL+` WHERE ($1 = '' OR e.estado::text = $1) ORDER BY e.created_at DESC`, estado)
}

func (s *Svc) EppDetalle(ctx context.Context, a *Actor, id int) (map[string]any, error) {
	row, err := One(ctx, s.DB, eppSQL+` WHERE e.id = $1`, id)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	json.Unmarshal(row, &m)
	if a.Rol != "sst" && int(m["puesto_id"].(float64)) != a.PuestoID {
		return nil, ErrNotFound
	}
	h, err := HistList(ctx, s.DB, "solicitud_epp", id)
	m["historial"] = h
	return m, err
}

type EppAccionIn struct {
	Fecha     string `json:"fecha"`
	Motivo    string `json:"motivo"`
	Devuelto  bool   `json:"epp_anterior_devuelto"`
	Firmado   bool   `json:"formato_firmado"`
}

// EppAccion ejecuta aprobar | espera | rechazar | entregar.
func (s *Svc) EppAccion(ctx context.Context, a *Actor, id int, accion string, in EppAccionIn) error {
	to := map[string]string{"aprobar": "aprobada", "espera": "en_espera", "rechazar": "rechazada", "entregar": "entregada"}[accion]
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var estado, recurso string
		var puestoID, recursoID, stock, minimo int
		err := tx.QueryRow(ctx, `SELECT e.estado::text, e.puesto_id, r.id, r.nombre, r.stock, r.stock_minimo
			FROM solicitudes_epp e JOIN recursos r ON r.id = e.recurso_id WHERE e.id = $1 FOR UPDATE OF e, r`, id).
			Scan(&estado, &puestoID, &recursoID, &recurso, &stock, &minimo)
		if err != nil {
			return notFound(err)
		}
		if !PuedeEPP(estado, to) {
			return errTrans()
		}
		code := Code("E", id)
		var hist, aviso string
		switch to {
		case "aprobada":
			if stock <= 0 {
				return Conflict("stock_insuficiente", "No hay stock para aprobar esta solicitud.")
			}
			_, err = tx.Exec(ctx, `UPDATE solicitudes_epp SET estado='aprobada' WHERE id=$1`, id)
			hist, aviso = "Aprobó la solicitud", fmt.Sprintf("Tu solicitud %s fue aprobada. Acércate a SST para la entrega.", code)
		case "en_espera":
			f, perr := time.Parse("2006-01-02", in.Fecha)
			if perr != nil {
				return Bad("fecha_requerida", "Indica la fecha estimada de cambio.")
			}
			_, err = tx.Exec(ctx, `UPDATE solicitudes_epp SET estado='en_espera', fecha_espera=$2 WHERE id=$1`, id, f)
			fd := f.Format("02/01/2006")
			hist = "Puso la solicitud en espera hasta el " + fd + " por falta de stock"
			aviso = fmt.Sprintf("Tu solicitud %s quedó en espera. Fecha estimada de cambio: %s.", code, fd)
		case "rechazada":
			if strings.TrimSpace(in.Motivo) == "" {
				return Bad("motivo_requerido", "Indica el motivo del rechazo.")
			}
			_, err = tx.Exec(ctx, `UPDATE solicitudes_epp SET estado='rechazada', motivo_rechazo=$2 WHERE id=$1`, id, in.Motivo)
			hist, aviso = "Rechazó la solicitud", fmt.Sprintf("Tu solicitud %s fue rechazada: %s", code, in.Motivo)
		case "entregada":
			if !in.Devuelto || !in.Firmado {
				return Bad("entrega_incompleta", "Confirma que el EPP anterior fue devuelto y que el formato fue firmado.")
			}
			if stock <= 0 {
				return Conflict("stock_insuficiente", "No hay stock para entregar esta solicitud.")
			}
			if _, err = tx.Exec(ctx, `UPDATE solicitudes_epp SET estado='entregada', epp_anterior_devuelto=true, formato_firmado=true, entregada_at=now() WHERE id=$1`, id); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE recursos SET stock = stock - 1 WHERE id=$1`, recursoID); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `INSERT INTO movimientos_inventario (recurso_id, tipo, cantidad, nota, solicitud_epp_id, usuario_id, puesto_id)
				VALUES ($1,'salida',1,$2,$3,$4,$5)`, recursoID, "Entrega de "+code, id, a.UserID, a.PuestoID); err != nil {
				return err
			}
			if err = Hist(ctx, tx, a, "recurso", recursoID, "Salida de 1 por entrega de "+code); err != nil {
				return err
			}
			if stock-1 <= minimo {
				if err = Notify(ctx, tx, "sst", 0, fmt.Sprintf("Stock bajo: %s (%d en stock, mínimo %d).", recurso, stock-1, minimo), "inventario", recursoID); err != nil {
					return err
				}
			}
			hist, aviso = "Marcó como entregada; EPP anterior devuelto y formato firmado", fmt.Sprintf("Tu solicitud %s fue entregada.", code)
		}
		if err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "solicitud_epp", id, hist); err != nil {
			return err
		}
		return Notify(ctx, tx, "trabajador", puestoID, aviso, "epp", id)
	})
}
