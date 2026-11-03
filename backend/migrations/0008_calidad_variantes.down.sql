ALTER TABLE movimientos_inventario DROP COLUMN IF EXISTS variante;
DROP TABLE IF EXISTS recurso_variantes;
UPDATE recursos SET tallas = NULL WHERE tallas = 'extintor';
ALTER TABLE recursos DROP CONSTRAINT IF EXISTS recursos_tallas_check;
ALTER TABLE recursos ADD CONSTRAINT recursos_tallas_check CHECK (tallas IN ('letras','calzado'));
UPDATE areas SET activa = false, puede_resolver_hallazgos = false WHERE nombre = 'Calidad';
