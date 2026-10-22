CREATE TYPE rol_usuario     AS ENUM ('trabajador','sst','mantenimiento','administracion');
CREATE TYPE nivel_riesgo    AS ENUM ('alto','medio','bajo');
CREATE TYPE prioridad       AS ENUM ('alta','media','baja');
CREATE TYPE estado_epp      AS ENUM ('enviada','aprobada','en_espera','rechazada','entregada');
CREATE TYPE estado_hallazgo AS ENUM ('registrado','en_atencion','cerrado','descartado');
CREATE TYPE estado_tarea    AS ENUM ('asignada','modificacion','en_proceso','por_validar','devuelta','cerrada');
CREATE TYPE estado_interna  AS ENUM ('pendiente','en_curso','completada');
CREATE TYPE tipo_contenido  AS ENUM ('afiche','flyer','procedimiento','instructivo');

CREATE TABLE areas (
  id     SERIAL PRIMARY KEY,
  nombre TEXT NOT NULL UNIQUE,
  tipo   TEXT NOT NULL CHECK (tipo IN ('operativa','administrativa')),
  puede_resolver_hallazgos BOOLEAN NOT NULL DEFAULT false,
  es_ubicacion BOOLEAN NOT NULL DEFAULT false
);
CREATE TABLE tipos_hallazgo (id SERIAL PRIMARY KEY, nombre TEXT NOT NULL UNIQUE);
CREATE TABLE categorias_recurso (id SERIAL PRIMARY KEY, nombre TEXT NOT NULL UNIQUE);

CREATE TABLE usuarios (
  id            SERIAL PRIMARY KEY,
  dni           CHAR(8) NOT NULL UNIQUE CHECK (dni ~ '^[0-9]{8}$'),
  nombres       TEXT NOT NULL,
  correo        TEXT NOT NULL,
  celular       TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  area_id       INT REFERENCES areas(id),
  estado        TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','activo','inactivo')),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE puestos (
  id      SERIAL PRIMARY KEY,
  nombre  TEXT NOT NULL,
  rol     rol_usuario NOT NULL,
  area_id INT NOT NULL REFERENCES areas(id)
);
CREATE TABLE asignaciones_puesto (
  id         SERIAL PRIMARY KEY,
  puesto_id  INT NOT NULL REFERENCES puestos(id),
  usuario_id INT NOT NULL REFERENCES usuarios(id),
  desde      TIMESTAMPTZ NOT NULL DEFAULT now(),
  hasta      TIMESTAMPTZ,
  asignado_por INT REFERENCES usuarios(id)
);
CREATE UNIQUE INDEX un_puesto_activo  ON asignaciones_puesto(puesto_id)  WHERE hasta IS NULL;
CREATE UNIQUE INDEX un_usuario_activo ON asignaciones_puesto(usuario_id) WHERE hasta IS NULL;

CREATE TABLE archivos (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  ruta       TEXT NOT NULL,
  mime       TEXT NOT NULL,
  bytes      INT  NOT NULL,
  subido_por INT  NOT NULL REFERENCES usuarios(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE recursos (
  id           SERIAL PRIMARY KEY,
  categoria_id INT NOT NULL REFERENCES categorias_recurso(id),
  nombre       TEXT NOT NULL,
  stock        INT NOT NULL DEFAULT 0 CHECK (stock >= 0),
  stock_minimo INT NOT NULL DEFAULT 0 CHECK (stock_minimo >= 0),
  activo       BOOLEAN NOT NULL DEFAULT true
);
CREATE TABLE movimientos_inventario (
  id         SERIAL PRIMARY KEY,
  recurso_id INT NOT NULL REFERENCES recursos(id),
  tipo       TEXT NOT NULL CHECK (tipo IN ('ingreso','salida')),
  cantidad   INT NOT NULL CHECK (cantidad > 0),
  nota       TEXT,
  solicitud_epp_id INT,
  usuario_id INT NOT NULL REFERENCES usuarios(id),
  puesto_id  INT NOT NULL REFERENCES puestos(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE solicitudes_epp (
  id              SERIAL PRIMARY KEY,
  puesto_id       INT NOT NULL REFERENCES puestos(id),
  usuario_id      INT NOT NULL REFERENCES usuarios(id),
  recurso_id      INT NOT NULL REFERENCES recursos(id),
  motivo          TEXT NOT NULL,
  descripcion     TEXT NOT NULL,
  foto_id         UUID NOT NULL REFERENCES archivos(id),
  estado          estado_epp NOT NULL DEFAULT 'enviada',
  fecha_espera    DATE,
  motivo_rechazo  TEXT,
  epp_anterior_devuelto BOOLEAN NOT NULL DEFAULT false,
  formato_firmado BOOLEAN NOT NULL DEFAULT false,
  entregada_at    TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE hallazgos (
  id                  SERIAL PRIMARY KEY,
  descripcion         TEXT NOT NULL,
  detalle             TEXT NOT NULL,
  tipo_id             INT NOT NULL REFERENCES tipos_hallazgo(id),
  ubicacion_id        INT NOT NULL REFERENCES areas(id),
  area_responsable_id INT NOT NULL REFERENCES areas(id),
  nivel_riesgo        nivel_riesgo NOT NULL,
  prioridad           prioridad NOT NULL,
  responsable_id      INT NOT NULL REFERENCES usuarios(id),
  estado              estado_hallazgo NOT NULL DEFAULT 'registrado',
  motivo_descarte     TEXT,
  cerrado_at          TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE hallazgo_fotos (
  hallazgo_id INT  NOT NULL REFERENCES hallazgos(id) ON DELETE CASCADE,
  archivo_id  UUID NOT NULL REFERENCES archivos(id),
  PRIMARY KEY (hallazgo_id, archivo_id)
);

CREATE TABLE tareas (
  id            SERIAL PRIMARY KEY,
  hallazgo_id   INT NOT NULL REFERENCES hallazgos(id),
  titulo        TEXT NOT NULL,
  indicaciones  TEXT NOT NULL,
  area_id       INT NOT NULL REFERENCES areas(id),
  puesto_id     INT NOT NULL REFERENCES puestos(id),
  creada_por    INT NOT NULL REFERENCES usuarios(id),
  fecha_limite  DATE NOT NULL,
  estado        estado_tarea NOT NULL DEFAULT 'asignada',
  devolucion_comentario TEXT,
  cerrada_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE tarea_pasos (
  id           SERIAL PRIMARY KEY,
  tarea_id     INT NOT NULL REFERENCES tareas(id) ON DELETE CASCADE,
  orden        INT NOT NULL,
  titulo       TEXT NOT NULL,
  completado   BOOLEAN NOT NULL DEFAULT false,
  rehacer      BOOLEAN NOT NULL DEFAULT false,
  evidencia_id UUID REFERENCES archivos(id),
  comentario   TEXT,
  completado_por INT REFERENCES usuarios(id),
  completado_at  TIMESTAMPTZ,
  UNIQUE (tarea_id, orden)
);
CREATE TABLE solicitudes_modificacion (
  id            SERIAL PRIMARY KEY,
  tarea_id      INT NOT NULL REFERENCES tareas(id),
  tipo          TEXT NOT NULL CHECK (tipo IN ('plazo','alcance')),
  nueva_fecha   DATE,
  motivo        TEXT NOT NULL,
  estado        TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','aprobada','rechazada')),
  respuesta     TEXT,
  solicitada_por INT NOT NULL REFERENCES usuarios(id),
  respondida_por INT REFERENCES usuarios(id),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  respondida_at TIMESTAMPTZ
);

CREATE TABLE tareas_internas (
  id SERIAL PRIMARY KEY, titulo TEXT NOT NULL, responsable TEXT NOT NULL,
  fecha DATE NOT NULL, estado estado_interna NOT NULL DEFAULT 'pendiente',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE contenidos_biblioteca (
  id SERIAL PRIMARY KEY, tipo tipo_contenido NOT NULL, codigo TEXT, titulo TEXT NOT NULL,
  resumen TEXT NOT NULL, cuerpo TEXT NOT NULL DEFAULT '', imagen_id UUID REFERENCES archivos(id),
  publicado_por INT NOT NULL REFERENCES usuarios(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  eliminado_at TIMESTAMPTZ
);

CREATE TABLE historial (
  id          BIGSERIAL PRIMARY KEY,
  entidad     TEXT NOT NULL,
  entidad_id  INT  NOT NULL,
  accion      TEXT NOT NULL,
  usuario_id  INT  NOT NULL REFERENCES usuarios(id),
  puesto_id   INT  NOT NULL REFERENCES puestos(id),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ON historial (entidad, entidad_id);
CREATE TABLE notificaciones (
  id              BIGSERIAL PRIMARY KEY,
  para_rol        rol_usuario NOT NULL,
  para_puesto_id  INT REFERENCES puestos(id),
  texto           TEXT NOT NULL,
  ref_tipo        TEXT,
  ref_id          INT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE notificaciones_leidas (
  notificacion_id BIGINT NOT NULL REFERENCES notificaciones(id) ON DELETE CASCADE,
  usuario_id      INT    NOT NULL REFERENCES usuarios(id),
  leida_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (notificacion_id, usuario_id)
);
