package service

import (
	"context"
	"encoding/json"
)

// Dashboard calcula todos los indicadores en Postgres y devuelve un solo objeto JSON.
func (s *Svc) Dashboard(ctx context.Context) (json.RawMessage, error) {
	var out json.RawMessage
	err := s.DB.QueryRow(ctx, `
WITH h AS (
  SELECT h.*, ar.nombre AS area_resp, ub.nombre AS ubicacion, t.nombre AS tipo
  FROM hallazgos h JOIN areas ar ON ar.id = h.area_responsable_id JOIN areas ub ON ub.id = h.ubicacion_id
  JOIN tipos_hallazgo t ON t.id = h.tipo_id WHERE h.estado <> 'descartado'
)
SELECT json_build_object(
  'abiertos',          (SELECT count(*) FROM h WHERE estado IN ('registrado','en_atencion')),
  'abiertos_alto',     (SELECT count(*) FROM h WHERE estado IN ('registrado','en_atencion') AND nivel_riesgo = 'alto'),
  'dias_promedio',     (SELECT round(avg(extract(epoch FROM cerrado_at - created_at) / 86400)::numeric, 1) FROM h WHERE cerrado_at IS NOT NULL),
  'epp_nuevas',        (SELECT count(*) FROM solicitudes_epp WHERE estado = 'enviada'),
  'tareas_por_validar',(SELECT count(*) FROM tareas WHERE estado = 'por_validar'),
  'modificaciones',    (SELECT count(*) FROM tareas WHERE estado = 'modificacion'),
  'bajo_minimo',       (SELECT coalesce(json_agg(json_build_object('id', id, 'nombre', nombre, 'stock', stock, 'minimo', stock_minimo)), '[]')
                         FROM recursos WHERE activo AND stock <= stock_minimo),
  'por_area', (SELECT coalesce(json_agg(x ORDER BY x.cantidad DESC), '[]') FROM (
      SELECT area_responsable_id AS id, area_resp AS nombre, count(*) AS cantidad,
        count(*) FILTER (WHERE estado IN ('registrado','en_atencion')) AS abiertos,
        round(100.0 * count(*) / sum(count(*)) OVER (), 0) AS porcentaje
      FROM h GROUP BY 1, 2) x),
  'por_tipo', (SELECT coalesce(json_agg(x ORDER BY x.cantidad DESC), '[]') FROM (SELECT tipo AS nombre, count(*) AS cantidad FROM h GROUP BY 1) x),
  'por_riesgo', (SELECT coalesce(json_agg(x), '[]') FROM (SELECT nivel_riesgo AS nombre, count(*) AS cantidad,
      round(avg(extract(epoch FROM cerrado_at - created_at) / 86400)::numeric, 1) AS dias_promedio FROM h GROUP BY 1 ORDER BY 1) x),
  'por_ubicacion', (SELECT coalesce(json_agg(x ORDER BY x.cantidad DESC), '[]') FROM (SELECT ubicacion AS nombre, count(*) AS cantidad FROM h GROUP BY 1) x),
  'por_mes', (SELECT coalesce(json_agg(x ORDER BY x.mes), '[]') FROM (
      SELECT to_char(m, 'YYYY-MM') AS mes,
        count(h.id) FILTER (WHERE h.nivel_riesgo = 'alto') AS alto,
        count(h.id) FILTER (WHERE h.nivel_riesgo = 'medio') AS medio,
        count(h.id) FILTER (WHERE h.nivel_riesgo = 'bajo') AS bajo
      FROM generate_series(date_trunc('month', now()) - interval '5 months', date_trunc('month', now()), interval '1 month') m
      LEFT JOIN h ON date_trunc('month', h.fecha) = m GROUP BY m) x)
)`).Scan(&out)
	return out, err
}
