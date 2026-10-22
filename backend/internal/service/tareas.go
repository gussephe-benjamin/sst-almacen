package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const tareaSQL = `SELECT t.id, 'T-' || lpad(t.id::text, 3, '0') AS codigo, t.titulo, t.indicaciones, t.estado, t.fecha_limite,
	t.devolucion_comentario, t.cerrada_at, t.created_at, t.hallazgo_id, 'H-' || lpad(t.hallazgo_id::text, 3, '0') AS hallazgo_codigo,
	h.descripcion AS hallazgo, ub.nombre AS ubicacion, a.nombre AS area, p.id AS puesto_id, p.nombre AS puesto,
	(SELECT u.nombres FROM asignaciones_puesto ap JOIN usuarios u ON u.id = ap.usuario_id WHERE ap.puesto_id = p.id AND ap.hasta IS NULL) AS responsable,
	(SELECT count(*) FROM tarea_pasos s WHERE s.tarea_id = t.id) AS pasos_total,
	(SELECT count(*) FROM tarea_pasos s WHERE s.tarea_id = t.id AND s.completado) AS pasos_hechos
	FROM tareas t JOIN hallazgos h ON h.id = t.hallazgo_id JOIN areas ub ON ub.id = h.ubicacion_id
	JOIN areas a ON a.id = t.area_id JOIN puestos p ON p.id = t.puesto_id`

type TareaIn struct {
	Titulo       string   `json:"titulo"`
	Indicaciones string   `json:"indicaciones"`
	PuestoID     int      `json:"puesto_id"`
	FechaLimite  string   `json:"fecha_limite"`
	Pasos        []string `json:"pasos"`
}

func (s *Svc) TareaCrear(ctx context.Context, a *Actor, hallazgoID int, in TareaIn) (int, error) {
	fecha, err := time.Parse("2006-01-02", in.FechaLimite)
	if strings.TrimSpace(in.Titulo) == "" || in.PuestoID == 0 || err != nil {
		return 0, Bad("datos_incompletos", "Completa el título, el responsable y la fecha límite.")
	}
	var pasos []string
	for _, p := range in.Pasos {
		if strings.TrimSpace(p) != "" {
			pasos = append(pasos, strings.TrimSpace(p))
		}
	}
	if len(pasos) == 0 {
		return 0, Bad("pasos_requeridos", "Agrega al menos un paso.")
	}
	var id int
	err = s.Tx(ctx, func(tx pgx.Tx) error {
		estado, _, _, _, err := s.hallazgoEstado(ctx, tx, hallazgoID)
		if err != nil {
			return err
		}
		if estado != "registrado" && estado != "en_atencion" {
			return Conflict("hallazgo_cerrado", "El hallazgo ya está cerrado o descartado.")
		}
		var rol string
		var areaID int
		if err := tx.QueryRow(ctx, `SELECT rol::text, area_id FROM puestos WHERE id=$1`, in.PuestoID).Scan(&rol, &areaID); err != nil {
			return notFound(err)
		}
		if rol != "mantenimiento" && rol != "administracion" {
			return Bad("puesto_invalido", "Las tareas se asignan a Mantenimiento o Administración.")
		}
		if err := tx.QueryRow(ctx, `INSERT INTO tareas (hallazgo_id, titulo, indicaciones, area_id, puesto_id, creada_por, fecha_limite)
			VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, hallazgoID, in.Titulo, in.Indicaciones, areaID, in.PuestoID, a.UserID, fecha).Scan(&id); err != nil {
			return err
		}
		for i, p := range pasos {
			if _, err := tx.Exec(ctx, `INSERT INTO tarea_pasos (tarea_id, orden, titulo) VALUES ($1,$2,$3)`, id, i+1, p); err != nil {
				return err
			}
		}
		if estado == "registrado" {
			if _, err := tx.Exec(ctx, `UPDATE hallazgos SET estado='en_atencion' WHERE id=$1`, hallazgoID); err != nil {
				return err
			}
		}
		if err := Hist(ctx, tx, a, "hallazgo", hallazgoID, "Creó la tarea "+Code("T", id)); err != nil {
			return err
		}
		var persona string
		tx.QueryRow(ctx, `SELECT u.nombres FROM asignaciones_puesto ap JOIN usuarios u ON u.id = ap.usuario_id
			WHERE ap.puesto_id = $1 AND ap.hasta IS NULL`, in.PuestoID).Scan(&persona)
		if persona == "" {
			persona = "el puesto vacante"
		}
		if err := Hist(ctx, tx, a, "tarea", id, "Creó la tarea y la asignó a "+persona); err != nil {
			return err
		}
		return Notify(ctx, tx, rol, in.PuestoID, fmt.Sprintf("%s te asignó %s: %s.", a.Nombre, Code("T", id), in.Titulo), "tarea", id)
	})
	return id, err
}

func (s *Svc) Tareas(ctx context.Context, a *Actor) (json.RawMessage, error) {
	return List(ctx, s.DB, tareaSQL+` WHERE ($1 OR t.puesto_id = $2) ORDER BY t.fecha_limite`, a.Rol == "sst", a.PuestoID)
}

func (s *Svc) TareaDetalle(ctx context.Context, a *Actor, id int) (map[string]any, error) {
	row, err := One(ctx, s.DB, tareaSQL+` WHERE t.id = $1 AND ($2 OR t.puesto_id = $3)`, id, a.Rol == "sst", a.PuestoID)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	json.Unmarshal(row, &m)
	if m["pasos"], err = List(ctx, s.DB, `SELECT s.orden, s.titulo, s.completado, s.rehacer, s.evidencia_id, s.comentario, s.completado_at,
		u.nombres AS completado_por FROM tarea_pasos s LEFT JOIN usuarios u ON u.id = s.completado_por WHERE s.tarea_id=$1 ORDER BY s.orden`, id); err != nil {
		return nil, err
	}
	if m["modificaciones"], err = List(ctx, s.DB, `SELECT id, tipo, nueva_fecha, motivo, estado, respuesta, created_at FROM solicitudes_modificacion
		WHERE tarea_id=$1 ORDER BY created_at DESC`, id); err != nil {
		return nil, err
	}
	m["historial"], err = HistList(ctx, s.DB, "tarea", id)
	return m, err
}

type tareaRow struct {
	estado, rol, titulo string
	puestoID            int
}

// lockTarea bloquea la tarea y valida el acceso: SST, o el puesto responsable si soloResponsable.
func lockTarea(ctx context.Context, tx pgx.Tx, a *Actor, id int, soloResponsable bool) (tareaRow, error) {
	var t tareaRow
	err := tx.QueryRow(ctx, `SELECT t.estado::text, t.puesto_id, p.rol::text, t.titulo FROM tareas t JOIN puestos p ON p.id = t.puesto_id
		WHERE t.id=$1 FOR UPDATE OF t`, id).Scan(&t.estado, &t.puestoID, &t.rol, &t.titulo)
	if err != nil {
		return t, notFound(err)
	}
	if soloResponsable && t.puestoID != a.PuestoID {
		if a.Rol == "sst" {
			return t, ErrForbidden
		}
		return t, ErrNotFound
	}
	if !soloResponsable && a.Rol != "sst" {
		return t, ErrForbidden
	}
	return t, nil
}

func setEstado(ctx context.Context, tx pgx.Tx, id int, from, to string) error {
	if !PuedeTarea(from, to) {
		return errTrans()
	}
	_, err := tx.Exec(ctx, `UPDATE tareas SET estado=$2 WHERE id=$1`, id, to)
	return err
}

func (s *Svc) TareaAceptar(ctx context.Context, a *Actor, id int) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, true)
		if err != nil {
			return err
		}
		if err := setEstado(ctx, tx, id, t.estado, "en_proceso"); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "tarea", id, "Aceptó la tarea"); err != nil {
			return err
		}
		return Notify(ctx, tx, "sst", 0, fmt.Sprintf("%s aceptó %s.", a.Nombre, Code("T", id)), "tarea", id)
	})
}

type ModIn struct {
	Tipo       string `json:"tipo"`
	NuevaFecha string `json:"nueva_fecha"`
	Motivo     string `json:"motivo"`
	Respuesta  string `json:"respuesta"`
}

func (s *Svc) TareaPedirModificacion(ctx context.Context, a *Actor, id int, in ModIn) error {
	if (in.Tipo != "plazo" && in.Tipo != "alcance") || strings.TrimSpace(in.Motivo) == "" {
		return Bad("datos_incompletos", "Indica el tipo de modificación y el motivo.")
	}
	var fecha any
	if in.Tipo == "plazo" {
		f, err := time.Parse("2006-01-02", in.NuevaFecha)
		if err != nil {
			return Bad("fecha_requerida", "Indica la nueva fecha propuesta.")
		}
		fecha = f
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, true)
		if err != nil {
			return err
		}
		if err := setEstado(ctx, tx, id, t.estado, "modificacion"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO solicitudes_modificacion (tarea_id, tipo, nueva_fecha, motivo, solicitada_por) VALUES ($1,$2,$3,$4,$5)`,
			id, in.Tipo, fecha, in.Motivo, a.UserID); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "tarea", id, "Solicitó modificación de "+in.Tipo); err != nil {
			return err
		}
		return Notify(ctx, tx, "sst", 0, fmt.Sprintf("%s solicitó modificación de %s para %s.", a.Nombre, in.Tipo, Code("T", id)), "tarea", id)
	})
}

func (s *Svc) TareaResponderModificacion(ctx context.Context, a *Actor, id int, aprobar bool, respuesta string) error {
	if !aprobar && strings.TrimSpace(respuesta) == "" {
		return Bad("respuesta_requerida", "Escribe la respuesta para el responsable.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, false)
		if err != nil {
			return err
		}
		var modID int
		var tipo string
		var nueva *time.Time
		if err := tx.QueryRow(ctx, `SELECT id, tipo, nueva_fecha FROM solicitudes_modificacion WHERE tarea_id=$1 AND estado='pendiente'
			ORDER BY created_at DESC LIMIT 1`, id).Scan(&modID, &tipo, &nueva); err != nil {
			return Conflict("sin_modificacion", "No hay una solicitud de modificación pendiente.")
		}
		to, est, accion := "asignada", "rechazada", "Rechazó la modificación"
		if aprobar {
			to, est, accion = "en_proceso", "aprobada", "Aprobó la modificación de "+tipo
		}
		if err := setEstado(ctx, tx, id, t.estado, to); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE solicitudes_modificacion SET estado=$2, respuesta=$3, respondida_por=$4, respondida_at=now() WHERE id=$1`,
			modID, est, respuesta, a.UserID); err != nil {
			return err
		}
		if aprobar && tipo == "plazo" && nueva != nil {
			if _, err := tx.Exec(ctx, `UPDATE tareas SET fecha_limite=$2 WHERE id=$1`, id, *nueva); err != nil {
				return err
			}
			accion += "; nueva fecha límite " + nueva.Format("02/01/2006")
		}
		if err := Hist(ctx, tx, a, "tarea", id, accion); err != nil {
			return err
		}
		return Notify(ctx, tx, t.rol, t.puestoID, fmt.Sprintf("SST %s tu solicitud de modificación de %s.", map[bool]string{true: "aprobó", false: "rechazó"}[aprobar], Code("T", id)), "tarea", id)
	})
}

type PasoIn struct {
	EvidenciaID string `json:"evidencia_id"`
	Comentario  string `json:"comentario"`
}

func (s *Svc) PasoCompletar(ctx context.Context, a *Actor, id, orden int, in PasoIn, completar bool) error {
	if completar && (in.EvidenciaID == "" || strings.TrimSpace(in.Comentario) == "") {
		return Bad("evidencia_requerida", "Cada paso necesita una foto y un comentario.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, true)
		if err != nil {
			return err
		}
		if t.estado != "en_proceso" && t.estado != "devuelta" {
			return errTrans()
		}
		var tag interface{ RowsAffected() int64 }
		if completar {
			tag, err = tx.Exec(ctx, `UPDATE tarea_pasos SET completado=true, rehacer=false, evidencia_id=$3, comentario=$4, completado_por=$5, completado_at=now()
				WHERE tarea_id=$1 AND orden=$2`, id, orden, in.EvidenciaID, in.Comentario, a.UserID)
		} else {
			tag, err = tx.Exec(ctx, `UPDATE tarea_pasos SET completado=false WHERE tarea_id=$1 AND orden=$2`, id, orden)
		}
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		accion := fmt.Sprintf("Completó el paso %d con evidencia", orden)
		if !completar {
			accion = fmt.Sprintf("Reabrió el paso %d", orden)
		}
		return Hist(ctx, tx, a, "tarea", id, accion)
	})
}

func (s *Svc) TareaTerminar(ctx context.Context, a *Actor, id int) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, true)
		if err != nil {
			return err
		}
		var faltan int
		tx.QueryRow(ctx, `SELECT count(*) FROM tarea_pasos WHERE tarea_id=$1 AND (NOT completado OR evidencia_id IS NULL OR coalesce(comentario,'') = '')`, id).Scan(&faltan)
		if faltan > 0 {
			return Conflict("pasos_incompletos", "Completa todos los pasos con foto y comentario antes de terminar.")
		}
		if err := setEstado(ctx, tx, id, t.estado, "por_validar"); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "tarea", id, "Marcó la tarea como terminada"); err != nil {
			return err
		}
		return Notify(ctx, tx, "sst", 0, fmt.Sprintf("%s terminó %s. Revisa la evidencia para cerrarla.", a.Nombre, Code("T", id)), "tarea", id)
	})
}

func (s *Svc) TareaCerrar(ctx context.Context, a *Actor, id int) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, false)
		if err != nil {
			return err
		}
		if err := setEstado(ctx, tx, id, t.estado, "cerrada"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE tareas SET cerrada_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "tarea", id, "Validó y cerró la tarea"); err != nil {
			return err
		}
		return Notify(ctx, tx, t.rol, t.puestoID, fmt.Sprintf("SST validó y cerró %s.", Code("T", id)), "tarea", id)
	})
}

type DevolverIn struct {
	Comentario string `json:"comentario"`
	Pasos      []int  `json:"pasos"`
}

func (s *Svc) TareaDevolver(ctx context.Context, a *Actor, id int, in DevolverIn) error {
	if strings.TrimSpace(in.Comentario) == "" || len(in.Pasos) == 0 {
		return Bad("datos_incompletos", "Escribe un comentario y marca los pasos a rehacer.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		t, err := lockTarea(ctx, tx, a, id, false)
		if err != nil {
			return err
		}
		if err := setEstado(ctx, tx, id, t.estado, "devuelta"); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE tareas SET devolucion_comentario=$2 WHERE id=$1`, id, in.Comentario); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE tarea_pasos SET completado=false, rehacer=true WHERE tarea_id=$1 AND orden = ANY($2)`, id, in.Pasos); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "tarea", id, "Devolvió la tarea: "+in.Comentario); err != nil {
			return err
		}
		return Notify(ctx, tx, t.rol, t.puestoID, fmt.Sprintf("SST devolvió %s: %s", Code("T", id), in.Comentario), "tarea", id)
	})
}
