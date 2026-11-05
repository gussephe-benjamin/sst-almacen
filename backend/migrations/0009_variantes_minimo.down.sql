ALTER TABLE puestos DROP COLUMN IF EXISTS activo;
UPDATE tipos_hallazgo SET activo = false WHERE nombre = 'Observación';
ALTER TABLE recurso_variantes DROP COLUMN IF EXISTS stock_minimo;
