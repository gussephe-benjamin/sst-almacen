package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PuedeCerrarHallazgo aplica la regla de cierre: un hallazgo con tareas solo se cierra
// cuando todas están cerradas; sin tareas, solo si el área responsable es SST o Calidad.
func PuedeCerrarHallazgo(estado, areaResp string, tareas, cerradas int) bool {
	if !PuedeHallazgo(estado, "cerrado") {
		return false
	}
	if tareas == 0 {
		return estado == "registrado" && (areaResp == "SST" || areaResp == "Calidad")
	}
	return tareas == cerradas
}

const hallazgoSQL = `SELECT h.id, 'H-' || lpad(h.id::text, 3, '0') AS codigo, h.descripcion, h.detalle, h.estado,
	h.nivel_riesgo, h.prioridad, h.motivo_descarte, h.cerrado_at, h.created_at,
	t.id AS tipo_id, t.nombre AS tipo, ub.id AS ubicacion_id, ub.nombre AS ubicacion,
	ar.id AS area_responsable_id, ar.nombre AS area_responsable, u.nombres AS responsable,
	(SELECT count(*) FROM tareas x WHERE x.hallazgo_id = h.id) AS tareas,
	(SELECT count(*) FROM tareas x WHERE x.hallazgo_id = h.id AND x.estado = 'cerrada') AS tareas_cerradas
	FROM hallazgos h JOIN tipos_hallazgo t ON t.id = h.tipo_id JOIN areas ub ON ub.id = h.ubicacion_id
	JOIN areas ar ON ar.id = h.area_responsable_id JOIN usuarios u ON u.id = h.responsable_id`

type HallazgoFiltro struct {
	Estado, Riesgo, Q string
	Ubicacion, Responsable int
}

const hallazgoWhere = ` WHERE ($1 = '' OR h.estado::text = $1) AND ($2 = '' OR h.nivel_riesgo::text = $2)
	AND ($3 = 0 OR h.ubicacion_id = $3) AND ($4 = 0 OR h.area_responsable_id = $4)
	AND ($5 = '' OR h.descripcion ILIKE '%' || $5 || '%' OR h.detalle ILIKE '%' || $5 || '%')
	ORDER BY h.created_at DESC`

func (f HallazgoFiltro) args() []any {
	return []any{f.Estado, f.Riesgo, f.Ubicacion, f.Responsable, f.Q}
}

func (s *Svc) Hallazgos(ctx context.Context, f HallazgoFiltro) (json.RawMessage, error) {
	return List(ctx, s.DB, hallazgoSQL+hallazgoWhere, f.args()...)
}

// HallazgosRows devuelve las filas filtradas para exportar.
func (s *Svc) HallazgosRows(ctx context.Context, f HallazgoFiltro) ([]map[string]any, error) {
	raw, err := s.Hallazgos(ctx, f)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	err = json.Unmarshal(raw, &rows)
	return rows, err
}

type HallazgoIn struct {
	Descripcion       string   `json:"descripcion"`
	Detalle           string   `json:"detalle"`
	TipoID            int      `json:"tipo_id"`
	UbicacionID       int      `json:"ubicacion_id"`
	AreaResponsableID int      `json:"area_responsable_id"`
	NivelRiesgo       string   `json:"nivel_riesgo"`
	Prioridad         string   `json:"prioridad"`
	Fotos             []string `json:"fotos"`
}

func (s *Svc) HallazgoCrear(ctx context.Context, a *Actor, in HallazgoIn) (int, error) {
	if strings.TrimSpace(in.Descripcion) == "" || in.TipoID == 0 || in.UbicacionID == 0 || in.AreaResponsableID == 0 || in.NivelRiesgo == "" || in.Prioridad == "" {
		return 0, Bad("datos_incompletos", "Completa la descripción, tipo, ubicación, área responsable, riesgo y prioridad.")
	}
	var id int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		var ok bool
		tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM areas WHERE id=$1 AND puede_resolver_hallazgos)`, in.AreaResponsableID).Scan(&ok)
		if !ok {
			return Bad("area_invalida", "El área responsable debe ser Mantenimiento, Administración, SST o Calidad.")
		}
		if err := tx.QueryRow(ctx, `INSERT INTO hallazgos (descripcion, detalle, tipo_id, ubicacion_id, area_responsable_id, nivel_riesgo, prioridad, responsable_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, in.Descripcion, in.Detalle, in.TipoID, in.UbicacionID, in.AreaResponsableID,
			in.NivelRiesgo, in.Prioridad, a.UserID).Scan(&id); err != nil {
			return err
		}
		for _, f := range in.Fotos {
			if _, err := tx.Exec(ctx, `INSERT INTO hallazgo_fotos (hallazgo_id, archivo_id) VALUES ($1,$2)`, id, f); err != nil {
				return err
			}
		}
		return Hist(ctx, tx, a, "hallazgo", id, "Registró el hallazgo durante la inspección")
	})
	return id, err
}

func (s *Svc) HallazgoDetalle(ctx context.Context, id int) (map[string]any, error) {
	row, err := One(ctx, s.DB, hallazgoSQL+` WHERE h.id = $1`, id)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	json.Unmarshal(row, &m)
	if m["fotos"], err = List(ctx, s.DB, `SELECT archivo_id AS id FROM hallazgo_fotos WHERE hallazgo_id=$1`, id); err != nil {
		return nil, err
	}
	if m["tareas_lista"], err = List(ctx, s.DB, tareaSQL+` WHERE t.hallazgo_id = $1 ORDER BY t.id`, id); err != nil {
		return nil, err
	}
	m["historial"], err = HistList(ctx, s.DB, "hallazgo", id)
	return m, err
}

func (s *Svc) hallazgoEstado(ctx context.Context, tx pgx.Tx, id int) (estado, area string, total, cerradas int, err error) {
	err = tx.QueryRow(ctx, `SELECT h.estado::text, a.nombre,
		(SELECT count(*) FROM tareas WHERE hallazgo_id=h.id), (SELECT count(*) FROM tareas WHERE hallazgo_id=h.id AND estado='cerrada')
		FROM hallazgos h JOIN areas a ON a.id = h.area_responsable_id WHERE h.id=$1 FOR UPDATE OF h`, id).Scan(&estado, &area, &total, &cerradas)
	return estado, area, total, cerradas, notFound(err)
}

func (s *Svc) HallazgoDescartar(ctx context.Context, a *Actor, id int, motivo string) error {
	if strings.TrimSpace(motivo) == "" {
		return Bad("motivo_requerido", "Indica el motivo del descarte.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		estado, _, total, cerradas, err := s.hallazgoEstado(ctx, tx, id)
		if err != nil {
			return err
		}
		if !PuedeHallazgo(estado, "descartado") {
			return errTrans()
		}
		if total != cerradas {
			return Conflict("tareas_activas", "No puedes descartar un hallazgo con tareas activas.")
		}
		if _, err := tx.Exec(ctx, `UPDATE hallazgos SET estado='descartado', motivo_descarte=$2 WHERE id=$1`, id, motivo); err != nil {
			return err
		}
		return Hist(ctx, tx, a, "hallazgo", id, "Descartó el hallazgo")
	})
}

func (s *Svc) HallazgoCerrar(ctx context.Context, a *Actor, id int) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		estado, area, total, cerradas, err := s.hallazgoEstado(ctx, tx, id)
		if err != nil {
			return err
		}
		if !PuedeCerrarHallazgo(estado, area, total, cerradas) {
			if total > cerradas {
				return Conflict("tareas_pendientes", "El hallazgo tiene tareas sin cerrar.")
			}
			if estado == "registrado" {
				return Conflict("requiere_tareas", "Este hallazgo debe resolverse con tareas al área responsable.")
			}
			return errTrans()
		}
		if _, err := tx.Exec(ctx, `UPDATE hallazgos SET estado='cerrado', cerrado_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		return Hist(ctx, tx, a, "hallazgo", id, "Cerró el hallazgo")
	})
}
