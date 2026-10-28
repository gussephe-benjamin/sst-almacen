-- Contraseña inicial que entregó el administrador, cifrada (AES-GCM). Se borra cuando la persona la cambia.
ALTER TABLE usuarios ADD COLUMN password_asignada TEXT;
