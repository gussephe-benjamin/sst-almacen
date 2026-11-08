package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const eppSQL = `SELECT e.id, 'E-' || lpad(e.id::text, 3, '0') AS codigo, e.estado, e.motivo, e.descripcion, e.talla, r.tallas,
	e.foto_id, e.fecha_espera, e.motivo_rechazo, e.epp_anterior_devuelto, e.formato_firmado, e.entregada_at, e.created_at,
	r.id AS recurso_id, r.nombre AS recurso, r.stock,
	(SELECT v.stock FROM recurso_variantes v WHERE v.recurso_id = r.id AND v.variante = e.talla) AS stock_talla, p.id AS puesto_id, p.nombre AS puesto, u.nombres AS trabajador, u.dni
	FROM solicitudes_epp e JOIN recursos r ON r.id = e.recurso_id JOIN puestos p ON p.id = e.puesto_id
	JOIN usuarios u ON u.id = e.usuario_id`

type EppIn struct {
	RecursoID   int    `json:"recurso_id"`
	Motivo      string `json:"motivo"`
	Talla       string `json:"talla"`
	Descripcion string `json:"descripcion"`
	FotoID      string `json:"foto_id"`
}

// Tallas (o tipos, en extintores) según el tipo de variante del recurso; sin tipo, es talla única.
// Cada variante tiene su propio stock en recurso_variantes.
var Tallas = map[string][]string{
	"letras":   {"XS", "S", "M", "L", "XL", "XXL"},
	"calzado":  {"35", "36", "37", "38", "39", "40", "41", "42", "43", "44", "45"},
	"extintor": {"PQS", "CO2", "Agua presurizada"},
}

// TallaValida comprueba la talla elegida para un EPP con tipo de talla (nil = talla única).
func TallaValida(tipo *string, talla string) bool {
	if tipo == nil {
		return talla == ""
	}
	for _, t := range Tallas[*tipo] {
		if t == talla {
			return true
		}
	}
	return false
}

func (s *Svc) EppCrear(ctx context.Context, a *Actor, in EppIn) (int, error) {
	if in.RecursoID == 0 || in.Motivo == "" {
		return 0, Bad("datos_incompletos", "Elige el EPP y el motivo.")
	}
	if in.FotoID == "" {
		return 0, Bad("foto_requerida", "Adjunta una foto del EPP dañado.")
	}
	var id int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var recurso string
		var tipo *string
		if err := tx.QueryRow(ctx, `SELECT nombre, tallas FROM recursos WHERE id=$1`, in.RecursoID).Scan(&recurso, &tipo); err != nil {
			return notFound(err)
		}
		if !TallaValida(tipo, in.Talla) {
			return Bad("talla_requerida", "Elige tu talla.")
		}
		var talla *string
		if in.Talla != "" {
			talla = &in.Talla
		}
		if err := tx.QueryRow(ctx, `INSERT INTO solicitudes_epp (puesto_id, usuario_id, recurso_id, motivo, talla, descripcion, foto_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, a.PuestoID, a.UserID, in.RecursoID, in.Motivo, talla, strings.TrimSpace(in.Descripcion), in.FotoID).Scan(&id); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "solicitud_epp", id, "Envió la solicitud"); err != nil {
			return err
		}
		if talla != nil {
			recurso += " talla " + *talla
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
	Fecha    string `json:"fecha"`
	Motivo   string `json:"motivo"`
	Devuelto bool   `json:"epp_anterior_devuelto"`
	Firmado  bool   `json:"formato_firmado"`
}

// EppAccion ejecuta aprobar | espera | rechazar | entregar.
func (s *Svc) EppAccion(ctx context.Context, a *Actor, id int, accion string, in EppAccionIn) error {
	to := map[string]string{"aprobar": "aprobada", "espera": "en_espera", "rechazar": "rechazada", "entregar": "entregada"}[accion]
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var estado, recurso, rolSolicitante string
		var talla *string
		var puestoID, recursoID, stock, minimo int
		err := tx.QueryRow(ctx, `SELECT e.estado::text, e.puesto_id, p.rol::text, e.talla, r.id, r.nombre, r.stock, r.stock_minimo
			FROM solicitudes_epp e JOIN recursos r ON r.id = e.recurso_id JOIN puestos p ON p.id = e.puesto_id
			WHERE e.id = $1 FOR UPDATE OF e, r`, id).
			Scan(&estado, &puestoID, &rolSolicitante, &talla, &recursoID, &recurso, &stock, &minimo)
		if err != nil {
			return notFound(err)
		}
		// Con talla, lo que cuenta es el stock y el mínimo de esa talla.
		disponible := stock
		if talla != nil {
			disponible, minimo = 0, 0
			tx.QueryRow(ctx, `SELECT stock, stock_minimo FROM recurso_variantes WHERE recurso_id=$1 AND variante=$2`, recursoID, *talla).Scan(&disponible, &minimo)
		}
		if !PuedeEPP(estado, to) {
			return errTrans()
		}
		code := Code("E", id)
		var hist, aviso string
		switch to {
		case "aprobada":
			if disponible <= 0 {
				return Conflict("stock_insuficiente", sinStock("aprobar", talla))
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
			if disponible <= 0 {
				return Conflict("stock_insuficiente", sinStock("entregar", talla))
			}
			if _, err = tx.Exec(ctx, `UPDATE solicitudes_epp SET estado='entregada', epp_anterior_devuelto=true, formato_firmado=true, entregada_at=now() WHERE id=$1`, id); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `UPDATE recursos SET stock = stock - 1 WHERE id=$1`, recursoID); err != nil {
				return err
			}
			if talla != nil {
				if _, err = tx.Exec(ctx, `UPDATE recurso_variantes SET stock = stock - 1 WHERE recurso_id=$1 AND variante=$2`, recursoID, *talla); err != nil {
					return err
				}
			}
			if _, err = tx.Exec(ctx, `INSERT INTO movimientos_inventario (recurso_id, tipo, cantidad, nota, solicitud_epp_id, usuario_id, puesto_id, variante)
				VALUES ($1,'salida',1,$2,$3,$4,$5,$6)`, recursoID, "Entrega de "+code, id, a.UserID, a.PuestoID, talla); err != nil {
				return err
			}
			if err = Hist(ctx, tx, a, "recurso", recursoID, "Salida de 1 por entrega de "+code); err != nil {
				return err
			}
			// Aviso de stock bajo: sin talla, contra el mínimo del recurso; con talla, contra el de esa talla (si tiene).
			if disponible-1 <= minimo && minimo > 0 {
				nombre := recurso
				if talla != nil {
					nombre += " " + *talla
				}
				if err = Notify(ctx, tx, "sst", 0, fmt.Sprintf("Stock bajo: %s (%d en stock, mínimo %d).", nombre, disponible-1, minimo), "inventario", recursoID); err != nil {
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
		// El aviso va al puesto que pidió el EPP: un trabajador o un área de apoyo.
		return Notify(ctx, tx, rolSolicitante, puestoID, aviso, "epp", id)
	})
}

func sinStock(accion string, talla *string) string {
	if talla != nil {
		return fmt.Sprintf("No hay stock de la talla %s para %s esta solicitud.", *talla, accion)
	}
	return fmt.Sprintf("No hay stock para %s esta solicitud.", accion)
}
