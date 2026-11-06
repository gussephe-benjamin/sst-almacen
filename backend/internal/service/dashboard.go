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
  SELECT h.*, ar.nombre AS area_resp, ub.nombre AS ubicacion, t.nombre AS tipo,
    -- Días de atención: desde la fecha del hallazgo hasta que se corrigió (su última tarea
    -- cerrada o, sin tareas, el cierre del hallazgo), en días calendario de Lima.
    CASE WHEN h.estado = 'cerrado' THEN
      (coalesce((SELECT max(ta.cerrada_at) FROM tareas ta WHERE ta.hallazgo_id = h.id AND ta.estado = 'cerrada'), h.cerrado_at)
        AT TIME ZONE 'America/Lima')::date - h.fecha END AS dias_atencion
  FROM hallazgos h JOIN areas ar ON ar.id = h.area_responsable_id JOIN areas ub ON ub.id = h.ubicacion_id
  JOIN tipos_hallazgo t ON t.id = h.tipo_id WHERE h.estado <> 'descartado'
)
SELECT json_build_object(
  'abiertos',          (SELECT count(*) FROM h WHERE estado IN ('registrado','en_atencion')),
  'abiertos_alto',     (SELECT count(*) FROM h WHERE estado IN ('registrado','en_atencion') AND nivel_riesgo = 'alto'),
  'total',             (SELECT count(*) FROM h),
  'cerrados',          (SELECT count(*) FROM h WHERE estado = 'cerrado'),
  'porcentaje_cerrados', (SELECT round(100.0 * count(*) FILTER (WHERE estado = 'cerrado') / nullif(count(*), 0), 0) FROM h),
  'dias_promedio',     (SELECT round(avg(dias_atencion), 1) FROM h),
  'epp_total',         (SELECT count(*) FROM solicitudes_epp),
  'epp_nuevas',        (SELECT count(*) FROM solicitudes_epp WHERE estado = 'enviada'),
  'epp_entregadas',    (SELECT count(*) FROM solicitudes_epp WHERE estado = 'entregada'),
  'tareas_total',      (SELECT count(*) FROM tareas),
  'tareas_por_validar',(SELECT count(*) FROM tareas WHERE estado = 'por_validar'),
  'tareas_cerradas',   (SELECT count(*) FROM tareas WHERE estado = 'cerrada'),
  'modificaciones',    (SELECT count(*) FROM tareas WHERE estado = 'modificacion'),
  -- Una fila por recurso sin tallas o por talla/tipo que llegó a su mínimo. Un mínimo 0 no avisa.
  'bajo_minimo',       (SELECT coalesce(json_agg(x ORDER BY x.nombre, x.variante), '[]') FROM (
      SELECT r.id, r.nombre, NULL AS variante, r.tallas, r.stock, r.stock_minimo AS minimo
      FROM recursos r WHERE r.activo AND r.tallas IS NULL AND r.stock_minimo > 0 AND r.stock <= r.stock_minimo
      UNION ALL
      SELECT r.id, r.nombre, v.variante, r.tallas, v.stock, v.stock_minimo
      FROM recursos r JOIN recurso_variantes v ON v.recurso_id = r.id
      WHERE r.activo AND r.tallas IS NOT NULL AND v.stock_minimo > 0 AND v.stock <= v.stock_minimo) x),
  'por_area', (SELECT coalesce(json_agg(x ORDER BY x.cantidad DESC), '[]') FROM (
      SELECT area_responsable_id AS id, area_resp AS nombre, count(*) AS cantidad,
        count(*) FILTER (WHERE estado IN ('registrado','en_atencion')) AS abiertos,
        round(100.0 * count(*) / sum(count(*)) OVER (), 0) AS porcentaje
      FROM h GROUP BY 1, 2) x),
  'por_tipo', (SELECT coalesce(json_agg(x ORDER BY x.cantidad DESC), '[]') FROM (SELECT tipo AS nombre, count(*) AS cantidad FROM h GROUP BY 1) x),
  'por_riesgo', (SELECT coalesce(json_agg(x), '[]') FROM (SELECT nivel_riesgo AS nombre, count(*) AS cantidad,
      round(avg(dias_atencion), 1) AS dias_promedio FROM h GROUP BY 1 ORDER BY 1) x),
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
