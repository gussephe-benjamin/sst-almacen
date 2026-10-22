package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ---------- Catálogos ----------

func (s *Svc) Catalogos(ctx context.Context) (map[string]json.RawMessage, error) {
	out := map[string]json.RawMessage{}
	qs := map[string]string{
		"areas":      `SELECT id, nombre, tipo, puede_resolver_hallazgos, es_ubicacion FROM areas ORDER BY id`,
		"tipos":      `SELECT id, nombre FROM tipos_hallazgo ORDER BY nombre`,
		"categorias": `SELECT id, nombre FROM categorias_recurso ORDER BY id`,
		"epp":        `SELECT r.id, r.nombre FROM recursos r JOIN categorias_recurso c ON c.id = r.categoria_id WHERE c.nombre = 'EPP' AND r.activo ORDER BY r.nombre`,
		"puestos_apoyo": `SELECT p.id, p.nombre, p.rol, u.nombres AS persona FROM puestos p
			LEFT JOIN asignaciones_puesto ap ON ap.puesto_id = p.id AND ap.hasta IS NULL LEFT JOIN usuarios u ON u.id = ap.usuario_id
			WHERE p.rol IN ('mantenimiento','administracion') ORDER BY p.id`,
	}
	for k, q := range qs {
		v, err := List(ctx, s.DB, q)
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}

func (s *Svc) AreasPublicas(ctx context.Context) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT id, nombre, tipo FROM areas ORDER BY id`)
}

// ---------- Biblioteca ----------

const biblioSQL = `SELECT id, 'B-' || lpad(id::text, 3, '0') AS ref, tipo, codigo, titulo, resumen, cuerpo, imagen_id, created_at, updated_at
	FROM contenidos_biblioteca WHERE eliminado_at IS NULL`

func (s *Svc) Biblioteca(ctx context.Context, tipo string) (json.RawMessage, error) {
	return List(ctx, s.DB, biblioSQL+` AND ($1 = '' OR tipo::text = $1) ORDER BY created_at DESC`, tipo)
}

func (s *Svc) BiblioDetalle(ctx context.Context, id int) (json.RawMessage, error) {
	return One(ctx, s.DB, biblioSQL+` AND id = $1`, id)
}

type BiblioIn struct {
	Tipo     string  `json:"tipo"`
	Codigo   string  `json:"codigo"`
	Titulo   string  `json:"titulo"`
	Resumen  string  `json:"resumen"`
	Cuerpo   string  `json:"cuerpo"`
	ImagenID *string `json:"imagen_id"`
}

func (in BiblioIn) valid() error {
	if in.Tipo == "" || strings.TrimSpace(in.Titulo) == "" || strings.TrimSpace(in.Resumen) == "" {
		return Bad("datos_incompletos", "Completa el tipo, el título y el resumen.")
	}
	return nil
}

func (s *Svc) BiblioGuardar(ctx context.Context, a *Actor, id int, in BiblioIn) (int, error) {
	if err := in.valid(); err != nil {
		return 0, err
	}
	if in.ImagenID != nil && *in.ImagenID == "" {
		in.ImagenID = nil
	}
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if id == 0 {
			if err := tx.QueryRow(ctx, `INSERT INTO contenidos_biblioteca (tipo, codigo, titulo, resumen, cuerpo, imagen_id, publicado_por)
				VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`, in.Tipo, in.Codigo, in.Titulo, in.Resumen, in.Cuerpo, in.ImagenID, a.UserID).Scan(&id); err != nil {
				return err
			}
			if err := Hist(ctx, tx, a, "biblioteca", id, "Publicó el contenido"); err != nil {
				return err
			}
			return Notify(ctx, tx, "trabajador", 0, "Nuevo en la biblioteca: "+in.Titulo+".", "biblioteca", id)
		}
		tag, err := tx.Exec(ctx, `UPDATE contenidos_biblioteca SET tipo=$2, codigo=$3, titulo=$4, resumen=$5, cuerpo=$6, imagen_id=$7, updated_at=now()
			WHERE id=$1 AND eliminado_at IS NULL`, id, in.Tipo, in.Codigo, in.Titulo, in.Resumen, in.Cuerpo, in.ImagenID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return Hist(ctx, tx, a, "biblioteca", id, "Editó el contenido")
	})
	return id, err
}

func (s *Svc) BiblioEliminar(ctx context.Context, a *Actor, id int) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE contenidos_biblioteca SET eliminado_at=now() WHERE id=$1 AND eliminado_at IS NULL`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return Hist(ctx, tx, a, "biblioteca", id, "Eliminó el contenido")
	})
}

// ---------- Tareas internas ----------

func (s *Svc) Internas(ctx context.Context) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT id, 'I-' || lpad(id::text, 3, '0') AS codigo, titulo, responsable, fecha, estado FROM tareas_internas ORDER BY fecha`)
}

type InternaIn struct {
	Titulo      string `json:"titulo"`
	Responsable string `json:"responsable"`
	Fecha       string `json:"fecha"`
	Estado      string `json:"estado"`
}

func (s *Svc) InternaGuardar(ctx context.Context, a *Actor, id int, in InternaIn) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		if id == 0 {
			f, err := time.Parse("2006-01-02", in.Fecha)
			if strings.TrimSpace(in.Titulo) == "" || err != nil {
				return Bad("datos_incompletos", "Completa el título y la fecha.")
			}
			if in.Responsable == "" {
				in.Responsable = a.Nombre
			}
			if err := tx.QueryRow(ctx, `INSERT INTO tareas_internas (titulo, responsable, fecha) VALUES ($1,$2,$3) RETURNING id`, in.Titulo, in.Responsable, f).Scan(&id); err != nil {
				return err
			}
			return Hist(ctx, tx, a, "tarea_interna", id, "Creó la tarea interna")
		}
		if in.Estado != "pendiente" && in.Estado != "en_curso" && in.Estado != "completada" {
			return Bad("estado_invalido", "Estado no válido.")
		}
		tag, err := tx.Exec(ctx, `UPDATE tareas_internas SET estado=$2 WHERE id=$1`, id, in.Estado)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return Hist(ctx, tx, a, "tarea_interna", id, "Cambió el estado a "+in.Estado)
	})
}

// ---------- Puestos ----------

func (s *Svc) Puestos(ctx context.Context, incluirAdmin bool) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT p.id, p.nombre, p.rol, a.id AS area_id, a.nombre AS area, u.id AS usuario_id, u.nombres AS persona, u.dni, u.celular, u.correo, ap.desde,
		(SELECT count(*) FROM asignaciones_puesto x WHERE x.puesto_id = p.id AND x.hasta IS NOT NULL) AS anteriores
		FROM puestos p JOIN areas a ON a.id = p.area_id
		LEFT JOIN asignaciones_puesto ap ON ap.puesto_id = p.id AND ap.hasta IS NULL LEFT JOIN usuarios u ON u.id = ap.usuario_id
		WHERE $1 OR p.rol <> 'admin' ORDER BY p.rol, p.id`, incluirAdmin)
}

func (s *Svc) Pendientes(ctx context.Context) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT u.id, u.dni, u.nombres, u.correo, u.celular, u.estado, a.nombre AS area, a.tipo, u.created_at
		FROM usuarios u LEFT JOIN areas a ON a.id = u.area_id WHERE u.estado IN ('pendiente','inactivo')
		ORDER BY u.estado = 'inactivo', u.created_at DESC`)
}

func (s *Svc) Asignar(ctx context.Context, a *Actor, puestoID, usuarioID int) error {
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var puesto, rol string
		if err := tx.QueryRow(ctx, `SELECT nombre, rol::text FROM puestos WHERE id=$1 FOR UPDATE`, puestoID).Scan(&puesto, &rol); err != nil {
			return notFound(err)
		}
		if !rolesGestionables[rol] {
			return Err(403, "sin_permiso", "Las cuentas de administrador se gestionan desde el archivo .env.")
		}
		var nombre, estado string
		if err := tx.QueryRow(ctx, `SELECT nombres, estado FROM usuarios WHERE id=$1 FOR UPDATE`, usuarioID).Scan(&nombre, &estado); err != nil {
			return notFound(err)
		}
		var ocupa bool
		tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM asignaciones_puesto WHERE usuario_id=$1 AND hasta IS NULL)`, usuarioID).Scan(&ocupa)
		if ocupa {
			return Conflict("ya_asignado", "Esa persona ya ocupa otro puesto.")
		}
		if err := liberar(ctx, tx, puestoID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO asignaciones_puesto (puesto_id, usuario_id, asignado_por) VALUES ($1,$2,$3)`, puestoID, usuarioID, a.UserID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE usuarios SET estado='activo' WHERE id=$1`, usuarioID); err != nil {
			return err
		}
		return Hist(ctx, tx, a, "puesto", puestoID, fmt.Sprintf("Asignó a %s al puesto %s", nombre, puesto))
	})
}

func (s *Svc) PuestoHistorial(ctx context.Context, id int) (map[string]json.RawMessage, error) {
	asig, err := List(ctx, s.DB, `SELECT u.nombres AS persona, u.dni, ap.desde, ap.hasta FROM asignaciones_puesto ap
		JOIN usuarios u ON u.id = ap.usuario_id WHERE ap.puesto_id=$1 ORDER BY ap.desde DESC`, id)
	if err != nil {
		return nil, err
	}
	acc, err := List(ctx, s.DB, `SELECT h.created_at AS fecha, h.accion, h.entidad, h.entidad_id, u.nombres AS quien
		FROM historial h JOIN usuarios u ON u.id = h.usuario_id WHERE h.puesto_id=$1 OR (h.entidad = 'puesto' AND h.entidad_id = $1) ORDER BY h.created_at DESC LIMIT 100`, id)
	return map[string]json.RawMessage{"asignaciones": asig, "acciones": acc}, err
}

// ---------- Notificaciones ----------

func (s *Svc) Notificaciones(ctx context.Context, a *Actor) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT n.id, n.texto, n.ref_tipo, n.ref_id, n.created_at, (l.usuario_id IS NOT NULL) AS leida
		FROM notificaciones n LEFT JOIN notificaciones_leidas l ON l.notificacion_id = n.id AND l.usuario_id = $3
		WHERE n.para_rol::text = $1 AND (n.para_puesto_id IS NULL OR n.para_puesto_id = $2) ORDER BY n.created_at DESC LIMIT 100`,
		a.Rol, a.PuestoID, a.UserID)
}

func (s *Svc) LeerNotificacion(ctx context.Context, a *Actor, id int64) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO notificaciones_leidas (notificacion_id, usuario_id)
		SELECT id, $2 FROM notificaciones WHERE ($1 = 0 OR id = $1) AND para_rol::text = $3 AND (para_puesto_id IS NULL OR para_puesto_id = $4)
		ON CONFLICT DO NOTHING`, id, a.UserID, a.Rol, a.PuestoID)
	return err
}

// ---------- Archivos ----------

// PuedeVerArchivo valida que el usuario pueda ver la entidad dueña del archivo.
func (s *Svc) PuedeVerArchivo(ctx context.Context, a *Actor, id string) (ruta, mime string, err error) {
	var ok bool
	err = s.DB.QueryRow(ctx, `SELECT f.ruta, f.mime, (
		$2 = 'sst' OR f.subido_por = $3
		OR EXISTS (SELECT 1 FROM solicitudes_epp e WHERE e.foto_id = f.id AND e.puesto_id = $4)
		OR EXISTS (SELECT 1 FROM contenidos_biblioteca b WHERE b.imagen_id = f.id AND b.eliminado_at IS NULL)
		OR EXISTS (SELECT 1 FROM tarea_pasos s JOIN tareas t ON t.id = s.tarea_id WHERE s.evidencia_id = f.id AND t.puesto_id = $4)
		OR EXISTS (SELECT 1 FROM hallazgo_fotos hf JOIN tareas t ON t.hallazgo_id = hf.hallazgo_id WHERE hf.archivo_id = f.id AND t.puesto_id = $4)
		) FROM archivos f WHERE f.id = $1`, id, a.Rol, a.UserID, a.PuestoID).Scan(&ruta, &mime, &ok)
	if err != nil || !ok {
		return "", "", ErrNotFound
	}
	return ruta, mime, nil
}

func (s *Svc) GuardarArchivo(ctx context.Context, a *Actor, ruta, mime string, bytes int) (string, error) {
	var id string
	err := s.DB.QueryRow(ctx, `INSERT INTO archivos (ruta, mime, bytes, subido_por) VALUES ($1,$2,$3,$4) RETURNING id::text`, ruta, mime, bytes, a.UserID).Scan(&id)
	return id, err
}
