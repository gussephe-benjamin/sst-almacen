-- Catálogos base que necesita la aplicación desde el primer día.
INSERT INTO areas (nombre, tipo, es_ubicacion, puede_resolver_hallazgos) VALUES
  ('Almacén A', 'operativa', true, false),
  ('Almacén B', 'operativa', true, false),
  ('Zona de carga', 'operativa', true, false),
  ('Patio de maniobras', 'operativa', true, false),
  ('Oficinas', 'administrativa', true, false),
  ('Mantenimiento', 'administrativa', false, true),
  ('Administración', 'administrativa', false, true),
  ('SST', 'administrativa', false, true),
  ('Calidad', 'administrativa', false, true),
  ('Dirección', 'administrativa', false, false)
ON CONFLICT (nombre) DO NOTHING;

INSERT INTO tipos_hallazgo (nombre) VALUES
  ('Almacenamiento y estanterías'), ('Orden y limpieza'), ('Equipos de emergencia'), ('Señalización'),
  ('Instalaciones eléctricas'), ('Tránsito de montacargas'), ('Infraestructura'), ('Uso de EPP'), ('Otro')
ON CONFLICT (nombre) DO NOTHING;

INSERT INTO categorias_recurso (nombre) VALUES
  ('EPP'), ('Extintores'), ('Señalización'), ('Botiquines'), ('Luces de emergencia')
ON CONFLICT (nombre) DO NOTHING;
