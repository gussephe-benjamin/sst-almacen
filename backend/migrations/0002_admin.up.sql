ALTER TYPE rol_usuario ADD VALUE IF NOT EXISTS 'admin';
INSERT INTO areas (nombre, tipo) VALUES ('Dirección', 'administrativa') ON CONFLICT (nombre) DO NOTHING;
