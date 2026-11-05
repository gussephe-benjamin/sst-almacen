-- Mínimo por talla o tipo, sin la variante "Sin talla", tipo "Observación" y puestos eliminables.
-- Es idempotente: `api reset` lo vuelve a ejecutar después de 0003, 0006 y 0008.

-- Cada talla o tipo tiene su propio stock mínimo (0 = sin aviso).
ALTER TABLE recurso_variantes ADD COLUMN IF NOT EXISTS stock_minimo INT NOT NULL DEFAULT 0 CHECK (stock_minimo >= 0);

-- Se quita la variante '' ("Sin talla") y el total vuelve a ser la suma de las tallas o tipos.
DELETE FROM recurso_variantes WHERE variante = '';
UPDATE recursos r SET stock = coalesce((SELECT sum(v.stock) FROM recurso_variantes v WHERE v.recurso_id = r.id), 0)
WHERE r.tallas IS NOT NULL;

INSERT INTO tipos_hallazgo (nombre, activo, orden) VALUES ('Observación', true, 7)
ON CONFLICT (nombre) DO UPDATE SET activo = true, orden = 7;

-- Un puesto con historial no se borra: se desactiva y deja de aparecer.
ALTER TABLE puestos ADD COLUMN IF NOT EXISTS activo BOOLEAN NOT NULL DEFAULT true;

-- Los avisos de registros nuevos son solo para el administrador.
DELETE FROM notificaciones WHERE para_rol = 'sst' AND ref_tipo = 'puestos' AND texto LIKE '%espera asignación de puesto.';
