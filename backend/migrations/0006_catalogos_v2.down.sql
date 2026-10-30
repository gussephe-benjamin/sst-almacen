-- Solo revierte el esquema; los catálogos nuevos se conservan.
ALTER TABLE solicitudes_epp ALTER COLUMN descripcion DROP DEFAULT;
ALTER TABLE solicitudes_epp DROP COLUMN IF EXISTS talla;
ALTER TABLE recursos DROP COLUMN IF EXISTS tallas;
ALTER TABLE hallazgos DROP COLUMN IF EXISTS fecha;
ALTER TABLE tipos_hallazgo DROP COLUMN IF EXISTS orden;
ALTER TABLE tipos_hallazgo DROP COLUMN IF EXISTS activo;
ALTER TABLE areas DROP COLUMN IF EXISTS orden;
ALTER TABLE areas DROP COLUMN IF EXISTS activa;
