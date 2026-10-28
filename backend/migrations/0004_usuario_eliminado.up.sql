-- Personas eliminadas por el administrador que ya dejaron historial: se ocultan sin borrar su rastro.
ALTER TABLE usuarios DROP CONSTRAINT usuarios_estado_check;
ALTER TABLE usuarios ADD CONSTRAINT usuarios_estado_check CHECK (estado IN ('pendiente','activo','inactivo','eliminado'));
