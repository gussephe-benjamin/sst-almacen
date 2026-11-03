-- Calidad es un área de apoyo, igual que Mantenimiento y Administración: recibe tareas de los hallazgos.
ALTER TYPE rol_usuario ADD VALUE IF NOT EXISTS 'calidad';
