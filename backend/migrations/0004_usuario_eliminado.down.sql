UPDATE usuarios SET estado = 'inactivo' WHERE estado = 'eliminado';
ALTER TABLE usuarios DROP CONSTRAINT usuarios_estado_check;
ALTER TABLE usuarios ADD CONSTRAINT usuarios_estado_check CHECK (estado IN ('pendiente','activo','inactivo'));
