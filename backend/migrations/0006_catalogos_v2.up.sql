-- Catálogos de la empresa (CAMBIOS.pdf): áreas, ubicaciones, tipos de hallazgo y EPP con tallas.
-- Es idempotente: `api reset` lo vuelve a ejecutar después de 0003_catalogos.

ALTER TABLE areas ADD COLUMN IF NOT EXISTS activa BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE areas ADD COLUMN IF NOT EXISTS orden INT NOT NULL DEFAULT 100;
ALTER TABLE tipos_hallazgo ADD COLUMN IF NOT EXISTS activo BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE tipos_hallazgo ADD COLUMN IF NOT EXISTS orden INT NOT NULL DEFAULT 100;

-- Fecha en que se encontró el hallazgo (puede ser anterior al registro).
ALTER TABLE hallazgos ADD COLUMN IF NOT EXISTS fecha DATE;
UPDATE hallazgos SET fecha = created_at::date WHERE fecha IS NULL;
ALTER TABLE hallazgos ALTER COLUMN fecha SET NOT NULL;
ALTER TABLE hallazgos ALTER COLUMN fecha SET DEFAULT current_date;

-- Tallas por EPP: 'letras' (S…XXL), 'calzado' (35…45) o NULL (talla única).
ALTER TABLE recursos ADD COLUMN IF NOT EXISTS tallas TEXT CHECK (tallas IN ('letras','calzado'));
ALTER TABLE solicitudes_epp ADD COLUMN IF NOT EXISTS talla TEXT;
ALTER TABLE solicitudes_epp ALTER COLUMN descripcion SET DEFAULT '';

-- Áreas de la empresa (todas pueden ser responsables de un hallazgo) y ubicaciones (sedes).
INSERT INTO areas (nombre, tipo, es_ubicacion, puede_resolver_hallazgos, activa, orden) VALUES
  ('SST', 'administrativa', false, true, true, 1),
  ('Administración', 'administrativa', false, true, true, 2),
  ('Comercial', 'administrativa', false, true, true, 3),
  ('Mantenimiento', 'operativa', false, true, true, 4),
  ('Almacén', 'operativa', false, true, true, 5),
  ('Transporte', 'operativa', false, true, true, 6),
  ('Vigilancia', 'operativa', false, true, true, 7),
  ('Sede Guardia Civil', 'operativa', true, false, true, 11),
  ('Sede Guardia Peruana 1465', 'operativa', true, false, true, 12),
  ('Sede Guardia Peruana 1479', 'operativa', true, false, true, 13),
  ('Laboratorio', 'operativa', true, false, true, 14)
ON CONFLICT (nombre) DO UPDATE SET es_ubicacion = EXCLUDED.es_ubicacion,
  puede_resolver_hallazgos = EXCLUDED.puede_resolver_hallazgos, activa = true, orden = EXCLUDED.orden;

-- Puestos y personas de áreas anteriores pasan a su área nueva.
UPDATE puestos SET area_id = (SELECT id FROM areas WHERE nombre = 'Almacén')
  WHERE area_id IN (SELECT id FROM areas WHERE nombre IN ('Almacén A','Almacén B','Zona de carga','Patio de maniobras'));
UPDATE usuarios SET area_id = (SELECT id FROM areas WHERE nombre = 'Almacén')
  WHERE area_id IN (SELECT id FROM areas WHERE nombre IN ('Almacén A','Almacén B','Zona de carga','Patio de maniobras'));
UPDATE puestos SET area_id = (SELECT id FROM areas WHERE nombre = 'Administración')
  WHERE area_id IN (SELECT id FROM areas WHERE nombre = 'Oficinas');
UPDATE usuarios SET area_id = (SELECT id FROM areas WHERE nombre = 'Administración')
  WHERE area_id IN (SELECT id FROM areas WHERE nombre = 'Oficinas');
UPDATE puestos SET area_id = (SELECT id FROM areas WHERE nombre = 'SST')
  WHERE area_id IN (SELECT id FROM areas WHERE nombre = 'Calidad');
UPDATE usuarios SET area_id = (SELECT id FROM areas WHERE nombre = 'SST')
  WHERE area_id IN (SELECT id FROM areas WHERE nombre = 'Calidad');

-- Lo anterior se oculta; se borra si nada lo usa. Dirección queda oculta: es el área del administrador.
UPDATE areas SET activa = false, puede_resolver_hallazgos = false WHERE nombre NOT IN
  ('SST','Administración','Comercial','Mantenimiento','Almacén','Transporte','Vigilancia',
   'Sede Guardia Civil','Sede Guardia Peruana 1465','Sede Guardia Peruana 1479','Laboratorio');
DELETE FROM areas a WHERE NOT a.activa AND a.nombre <> 'Dirección'
  AND NOT EXISTS (SELECT 1 FROM usuarios x WHERE x.area_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM puestos x WHERE x.area_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM tareas x WHERE x.area_id = a.id)
  AND NOT EXISTS (SELECT 1 FROM hallazgos x WHERE x.ubicacion_id = a.id OR x.area_responsable_id = a.id);

-- Tipos de hallazgo.
INSERT INTO tipos_hallazgo (nombre, activo, orden) VALUES
  ('Condición insegura', true, 1),
  ('Acto inseguro', true, 2),
  ('Incidente / cuasiaccidente', true, 3),
  ('Incumplimiento documental', true, 4),
  ('Oportunidad de mejora', true, 5),
  ('Hallazgo positivo', true, 6)
ON CONFLICT (nombre) DO UPDATE SET activo = true, orden = EXCLUDED.orden;
UPDATE tipos_hallazgo SET activo = false WHERE nombre NOT IN
  ('Condición insegura','Acto inseguro','Incidente / cuasiaccidente','Incumplimiento documental','Oportunidad de mejora','Hallazgo positivo');
DELETE FROM tipos_hallazgo t WHERE NOT t.activo AND NOT EXISTS (SELECT 1 FROM hallazgos x WHERE x.tipo_id = t.id);

-- EPP que se pueden solicitar (stock 0: SST registra los ingresos).
INSERT INTO recursos (categoria_id, nombre, tallas)
SELECT c.id, v.nombre, v.tallas FROM categorias_recurso c, (VALUES
  ('Botas de seguridad', 'calzado'),
  ('Chaleco reflectivo', 'letras'),
  ('Guantes de seguridad', 'letras'),
  ('Casco de seguridad', NULL::text),
  ('Tafilete', NULL::text),
  ('Tapones auditivos', NULL::text),
  ('Mascarilla', NULL::text),
  ('Faja', 'letras'),
  ('Protector solar', NULL::text),
  ('Barbiquejo', NULL::text)
) AS v(nombre, tallas)
WHERE c.nombre = 'EPP'
  AND NOT EXISTS (SELECT 1 FROM recursos r WHERE r.categoria_id = c.id AND lower(r.nombre) = lower(v.nombre));

-- El trabajador solo recibe avisos de sus EPP.
DELETE FROM notificaciones WHERE para_rol = 'trabajador' AND ref_tipo = 'biblioteca';
