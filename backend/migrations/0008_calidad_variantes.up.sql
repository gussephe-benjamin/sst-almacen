-- Área Calidad y stock por variante (talla de EPP o tipo de extintor).
-- Es idempotente: `api reset` lo vuelve a ejecutar después de 0003 y 0006.

INSERT INTO areas (nombre, tipo, es_ubicacion, puede_resolver_hallazgos, activa, orden)
VALUES ('Calidad', 'administrativa', false, true, true, 8)
ON CONFLICT (nombre) DO UPDATE SET es_ubicacion = false, puede_resolver_hallazgos = true, activa = true, orden = 8;

-- Tipos de variante: 'letras' (XS…XXL), 'calzado' (35…45), 'extintor' (PQS, CO2, Agua presurizada).
ALTER TABLE recursos DROP CONSTRAINT IF EXISTS recursos_tallas_check;
ALTER TABLE recursos ADD CONSTRAINT recursos_tallas_check CHECK (tallas IN ('letras','calzado','extintor'));

-- Stock de cada variante. recursos.stock sigue siendo el total (la suma de sus variantes).
CREATE TABLE IF NOT EXISTS recurso_variantes (
  recurso_id INT  NOT NULL REFERENCES recursos(id),
  variante   TEXT NOT NULL,
  stock      INT  NOT NULL DEFAULT 0 CHECK (stock >= 0),
  PRIMARY KEY (recurso_id, variante)
);
ALTER TABLE movimientos_inventario ADD COLUMN IF NOT EXISTS variante TEXT;

-- Recursos con variantes que aún no las tienen: se crean todas en 0.
INSERT INTO recurso_variantes (recurso_id, variante)
SELECT r.id, v FROM recursos r, unnest(CASE r.tallas
    WHEN 'letras' THEN ARRAY['XS','S','M','L','XL','XXL']
    WHEN 'calzado' THEN ARRAY['35','36','37','38','39','40','41','42','43','44','45']
    WHEN 'extintor' THEN ARRAY['PQS','CO2','Agua presurizada']
  END) AS v
WHERE r.tallas IS NOT NULL
ON CONFLICT DO NOTHING;

-- El stock que ya tenían esos recursos (sin talla) queda como variante '' ("Sin talla") para no perderlo.
INSERT INTO recurso_variantes (recurso_id, variante, stock)
SELECT r.id, '', r.stock - coalesce((SELECT sum(x.stock) FROM recurso_variantes x WHERE x.recurso_id = r.id), 0)
FROM recursos r
WHERE r.tallas IS NOT NULL
  AND r.stock > coalesce((SELECT sum(x.stock) FROM recurso_variantes x WHERE x.recurso_id = r.id), 0)
ON CONFLICT (recurso_id, variante) DO UPDATE SET stock = recurso_variantes.stock + EXCLUDED.stock;
