-- Postgres no permite quitar un valor de un ENUM; solo se elimina el área si no está en uso.
DELETE FROM areas WHERE nombre = 'Dirección' AND NOT EXISTS (SELECT 1 FROM puestos p WHERE p.area_id = areas.id);
