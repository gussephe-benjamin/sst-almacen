\# Plan: versión básica de SST Almacén



Objetivo: llevar el prototipo navegable (`docs/mvp-referencia.html`) a una aplicación real con base de datos, autenticación y permisos, manteniendo el mismo alcance funcional.



\## Alcance de la versión básica



\*\*Incluye:\*\* registro con DNI y asignación a puestos, solicitudes de EPP con foto, inventario con alertas de mínimo, hallazgos con área responsable, tareas a Mantenimiento y Administración con pasos y evidencia, solicitudes de modificación, validación y devolución, tareas internas de SST, biblioteca, notificaciones dentro de la app, dashboard y exportación de hallazgos a Excel y PDF.



\*\*No incluye todavía:\*\* notificaciones por correo o push, modo sin conexión, vista propia para Calidad (SST cierra esos hallazgos), almacenamiento en la nube para las fotos, firma digital de la entrega de EPP (sigue en formato físico).



\## Estrategia



Se construye por rebanadas verticales: primero la base común (repo, base de datos, autenticación y layout), y después cada flujo completo de punta a punta (base, API y pantallas) antes de pasar al siguiente. Así cada fase termina con algo que se puede probar contra el prototipo.



| Fase | Entrega | Se prueba cuando |

|---|---|---|

| 0 | Repo, Docker, Makefile | `make db-up` y `make api` responden |

| 1 | Esquema y datos de demostración | `make migrate \&\& make seed` sin errores |

| 2 | Núcleo del backend y autenticación | Login con DNI devuelve sesión y `/me` responde según el rol |

| 3 | Base del frontend | Cada rol entra y ve su menú |

| 4 | Ciclo EPP e inventario | Un trabajador pide un EPP, SST lo entrega y el stock baja |

| 5 | Ciclo hallazgo y tareas | Un hallazgo pasa por tarea, evidencia, devolución y cierre |

| 6 | Biblioteca, tareas internas, puestos y notificaciones | Un registro nuevo se asigna a un puesto y hereda su historial |

| 7 | Dashboard y exportación | Los indicadores coinciden con los datos y se descarga el Excel |

| 8 | Pruebas de punta a punta y despliegue | Los dos ciclos pasan en Playwright |



\---



\## Modelo de datos



Este es el punto de partida para la primera migración. Claude Code puede ajustar detalles, pero no las reglas de `CLAUDE.md`.



```sql

\-- Catálogos

CREATE TYPE rol\_usuario     AS ENUM ('trabajador','sst','mantenimiento','administracion');

CREATE TYPE nivel\_riesgo    AS ENUM ('alto','medio','bajo');

CREATE TYPE prioridad       AS ENUM ('alta','media','baja');

CREATE TYPE estado\_epp      AS ENUM ('enviada','aprobada','en\_espera','rechazada','entregada');

CREATE TYPE estado\_hallazgo AS ENUM ('registrado','en\_atencion','cerrado','descartado');

CREATE TYPE estado\_tarea    AS ENUM ('asignada','modificacion','en\_proceso','por\_validar','devuelta','cerrada');

CREATE TYPE estado\_interna  AS ENUM ('pendiente','en\_curso','completada');

CREATE TYPE tipo\_contenido  AS ENUM ('afiche','flyer','procedimiento','instructivo');



CREATE TABLE areas (

&#x20; id     SERIAL PRIMARY KEY,

&#x20; nombre TEXT NOT NULL UNIQUE,                      -- Almacén A, Mantenimiento, Calidad, SST...

&#x20; tipo   TEXT NOT NULL CHECK (tipo IN ('operativa','administrativa')),

&#x20; puede\_resolver\_hallazgos BOOLEAN NOT NULL DEFAULT false,  -- Mantenimiento, Administración, SST, Calidad

&#x20; es\_ubicacion BOOLEAN NOT NULL DEFAULT false       -- Almacén A, Zona de carga, Patio...

);

CREATE TABLE tipos\_hallazgo (id SERIAL PRIMARY KEY, nombre TEXT NOT NULL UNIQUE);

CREATE TABLE categorias\_recurso (id SERIAL PRIMARY KEY, nombre TEXT NOT NULL UNIQUE);



\-- Personas y puestos

CREATE TABLE usuarios (

&#x20; id            SERIAL PRIMARY KEY,

&#x20; dni           CHAR(8) NOT NULL UNIQUE CHECK (dni \~ '^\[0-9]{8}$'),

&#x20; nombres       TEXT NOT NULL,

&#x20; correo        TEXT NOT NULL,

&#x20; celular       TEXT NOT NULL,

&#x20; password\_hash TEXT NOT NULL,

&#x20; area\_id       INT REFERENCES areas(id),

&#x20; estado        TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','activo','inactivo')),

&#x20; created\_at    TIMESTAMPTZ NOT NULL DEFAULT now()

);

CREATE TABLE puestos (

&#x20; id      SERIAL PRIMARY KEY,

&#x20; nombre  TEXT NOT NULL,

&#x20; rol     rol\_usuario NOT NULL,

&#x20; area\_id INT NOT NULL REFERENCES areas(id)

);

CREATE TABLE asignaciones\_puesto (

&#x20; id         SERIAL PRIMARY KEY,

&#x20; puesto\_id  INT NOT NULL REFERENCES puestos(id),

&#x20; usuario\_id INT NOT NULL REFERENCES usuarios(id),

&#x20; desde      TIMESTAMPTZ NOT NULL DEFAULT now(),

&#x20; hasta      TIMESTAMPTZ,

&#x20; asignado\_por INT REFERENCES usuarios(id)

);

CREATE UNIQUE INDEX un\_puesto\_activo  ON asignaciones\_puesto(puesto\_id)  WHERE hasta IS NULL;

CREATE UNIQUE INDEX un\_usuario\_activo ON asignaciones\_puesto(usuario\_id) WHERE hasta IS NULL;



\-- Archivos (fotos e imágenes)

CREATE TABLE archivos (

&#x20; id         UUID PRIMARY KEY DEFAULT gen\_random\_uuid(),

&#x20; ruta       TEXT NOT NULL,

&#x20; mime       TEXT NOT NULL,

&#x20; bytes      INT  NOT NULL,

&#x20; subido\_por INT  NOT NULL REFERENCES usuarios(id),

&#x20; created\_at TIMESTAMPTZ NOT NULL DEFAULT now()

);



\-- Inventario

CREATE TABLE recursos (

&#x20; id           SERIAL PRIMARY KEY,

&#x20; categoria\_id INT NOT NULL REFERENCES categorias\_recurso(id),

&#x20; nombre       TEXT NOT NULL,

&#x20; stock        INT NOT NULL DEFAULT 0 CHECK (stock >= 0),

&#x20; stock\_minimo INT NOT NULL DEFAULT 0 CHECK (stock\_minimo >= 0),

&#x20; activo       BOOLEAN NOT NULL DEFAULT true

);

CREATE TABLE movimientos\_inventario (

&#x20; id         SERIAL PRIMARY KEY,

&#x20; recurso\_id INT NOT NULL REFERENCES recursos(id),

&#x20; tipo       TEXT NOT NULL CHECK (tipo IN ('ingreso','salida')),

&#x20; cantidad   INT NOT NULL CHECK (cantidad > 0),

&#x20; nota       TEXT,

&#x20; solicitud\_epp\_id INT,

&#x20; usuario\_id INT NOT NULL REFERENCES usuarios(id),

&#x20; puesto\_id  INT NOT NULL REFERENCES puestos(id),

&#x20; created\_at TIMESTAMPTZ NOT NULL DEFAULT now()

);



\-- Solicitudes de EPP

CREATE TABLE solicitudes\_epp (

&#x20; id              SERIAL PRIMARY KEY,

&#x20; puesto\_id       INT NOT NULL REFERENCES puestos(id),     -- puesto solicitante

&#x20; usuario\_id      INT NOT NULL REFERENCES usuarios(id),    -- persona que la envió

&#x20; recurso\_id      INT NOT NULL REFERENCES recursos(id),

&#x20; motivo          TEXT NOT NULL,

&#x20; descripcion     TEXT NOT NULL,

&#x20; foto\_id         UUID NOT NULL REFERENCES archivos(id),

&#x20; estado          estado\_epp NOT NULL DEFAULT 'enviada',

&#x20; fecha\_espera    DATE,

&#x20; motivo\_rechazo  TEXT,

&#x20; epp\_anterior\_devuelto BOOLEAN NOT NULL DEFAULT false,

&#x20; formato\_firmado BOOLEAN NOT NULL DEFAULT false,

&#x20; entregada\_at    TIMESTAMPTZ,

&#x20; created\_at      TIMESTAMPTZ NOT NULL DEFAULT now()

);



\-- Hallazgos

CREATE TABLE hallazgos (

&#x20; id                  SERIAL PRIMARY KEY,

&#x20; descripcion         TEXT NOT NULL,

&#x20; detalle             TEXT NOT NULL,

&#x20; tipo\_id             INT NOT NULL REFERENCES tipos\_hallazgo(id),

&#x20; ubicacion\_id        INT NOT NULL REFERENCES areas(id),

&#x20; area\_responsable\_id INT NOT NULL REFERENCES areas(id),

&#x20; nivel\_riesgo        nivel\_riesgo NOT NULL,

&#x20; prioridad           prioridad NOT NULL,

&#x20; responsable\_id      INT NOT NULL REFERENCES usuarios(id),  -- persona de SST que hace seguimiento

&#x20; estado              estado\_hallazgo NOT NULL DEFAULT 'registrado',

&#x20; motivo\_descarte     TEXT,

&#x20; cerrado\_at          TIMESTAMPTZ,

&#x20; created\_at          TIMESTAMPTZ NOT NULL DEFAULT now()

);

CREATE TABLE hallazgo\_fotos (

&#x20; hallazgo\_id INT  NOT NULL REFERENCES hallazgos(id) ON DELETE CASCADE,

&#x20; archivo\_id  UUID NOT NULL REFERENCES archivos(id),

&#x20; PRIMARY KEY (hallazgo\_id, archivo\_id)

);



\-- Tareas a áreas

CREATE TABLE tareas (

&#x20; id            SERIAL PRIMARY KEY,

&#x20; hallazgo\_id   INT NOT NULL REFERENCES hallazgos(id),

&#x20; titulo        TEXT NOT NULL,

&#x20; indicaciones  TEXT NOT NULL,

&#x20; area\_id       INT NOT NULL REFERENCES areas(id),

&#x20; puesto\_id     INT NOT NULL REFERENCES puestos(id),     -- puesto responsable

&#x20; creada\_por    INT NOT NULL REFERENCES usuarios(id),

&#x20; fecha\_limite  DATE NOT NULL,

&#x20; estado        estado\_tarea NOT NULL DEFAULT 'asignada',

&#x20; devolucion\_comentario TEXT,

&#x20; cerrada\_at    TIMESTAMPTZ,

&#x20; created\_at    TIMESTAMPTZ NOT NULL DEFAULT now()

);

CREATE TABLE tarea\_pasos (

&#x20; id           SERIAL PRIMARY KEY,

&#x20; tarea\_id     INT NOT NULL REFERENCES tareas(id) ON DELETE CASCADE,

&#x20; orden        INT NOT NULL,

&#x20; titulo       TEXT NOT NULL,

&#x20; completado   BOOLEAN NOT NULL DEFAULT false,

&#x20; rehacer      BOOLEAN NOT NULL DEFAULT false,

&#x20; evidencia\_id UUID REFERENCES archivos(id),

&#x20; comentario   TEXT,

&#x20; completado\_por INT REFERENCES usuarios(id),

&#x20; completado\_at  TIMESTAMPTZ,

&#x20; UNIQUE (tarea\_id, orden)

);

CREATE TABLE solicitudes\_modificacion (

&#x20; id            SERIAL PRIMARY KEY,

&#x20; tarea\_id      INT NOT NULL REFERENCES tareas(id),

&#x20; tipo          TEXT NOT NULL CHECK (tipo IN ('plazo','alcance')),

&#x20; nueva\_fecha   DATE,

&#x20; motivo        TEXT NOT NULL,

&#x20; estado        TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','aprobada','rechazada')),

&#x20; respuesta     TEXT,

&#x20; solicitada\_por INT NOT NULL REFERENCES usuarios(id),

&#x20; respondida\_por INT REFERENCES usuarios(id),

&#x20; created\_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

&#x20; respondida\_at TIMESTAMPTZ

);



\-- SST interno y biblioteca

CREATE TABLE tareas\_internas (

&#x20; id SERIAL PRIMARY KEY, titulo TEXT NOT NULL, responsable TEXT NOT NULL,

&#x20; fecha DATE NOT NULL, estado estado\_interna NOT NULL DEFAULT 'pendiente',

&#x20; created\_at TIMESTAMPTZ NOT NULL DEFAULT now()

);

CREATE TABLE contenidos\_biblioteca (

&#x20; id SERIAL PRIMARY KEY, tipo tipo\_contenido NOT NULL, codigo TEXT, titulo TEXT NOT NULL,

&#x20; resumen TEXT NOT NULL, cuerpo TEXT NOT NULL DEFAULT '', imagen\_id UUID REFERENCES archivos(id),

&#x20; publicado\_por INT NOT NULL REFERENCES usuarios(id),

&#x20; created\_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated\_at TIMESTAMPTZ NOT NULL DEFAULT now(),

&#x20; eliminado\_at TIMESTAMPTZ

);



\-- Trazabilidad y avisos

CREATE TABLE historial (

&#x20; id          BIGSERIAL PRIMARY KEY,

&#x20; entidad     TEXT NOT NULL,          -- 'solicitud\_epp' | 'hallazgo' | 'tarea' | 'recurso' | 'puesto'

&#x20; entidad\_id  INT  NOT NULL,

&#x20; accion      TEXT NOT NULL,          -- texto legible: "Aprobó la solicitud"

&#x20; usuario\_id  INT  NOT NULL REFERENCES usuarios(id),

&#x20; puesto\_id   INT  NOT NULL REFERENCES puestos(id),

&#x20; created\_at  TIMESTAMPTZ NOT NULL DEFAULT now()

);

CREATE INDEX ON historial (entidad, entidad\_id);

CREATE TABLE notificaciones (

&#x20; id              BIGSERIAL PRIMARY KEY,

&#x20; para\_rol        rol\_usuario NOT NULL,

&#x20; para\_puesto\_id  INT REFERENCES puestos(id),   -- NULL = todos los puestos de ese rol

&#x20; texto           TEXT NOT NULL,

&#x20; ref\_tipo        TEXT,

&#x20; ref\_id          INT,

&#x20; created\_at      TIMESTAMPTZ NOT NULL DEFAULT now()

);

CREATE TABLE notificaciones\_leidas (

&#x20; notificacion\_id BIGINT NOT NULL REFERENCES notificaciones(id) ON DELETE CASCADE,

&#x20; usuario\_id      INT    NOT NULL REFERENCES usuarios(id),

&#x20; leida\_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

&#x20; PRIMARY KEY (notificacion\_id, usuario\_id)

);

```



Las notificaciones apuntan a un rol o a un puesto, no a una persona, para que el reemplazo de un puesto vea los avisos pendientes. Lo leído sí se guarda por persona.



\## API



Todas las rutas van bajo `/api`. La sesión es un JWT en cookie `httpOnly`. Entre paréntesis, los roles permitidos.



\*\*Autenticación y archivos\*\*

\- `POST /auth/register` (público): DNI, nombres, correo, celular, área, contraseña. Queda `pendiente`.

\- `POST /auth/login` (público), `POST /auth/logout`, `GET /me` (usuario, puesto y rol actuales).

\- `POST /files` (todos, multipart, máximo 5 MB, solo imágenes) devuelve el `id`. `GET /files/{id}` valida que el usuario pueda ver la entidad dueña del archivo.



\*\*EPP\*\*

\- `GET /epp/mias`, `POST /epp` (trabajador)

\- `GET /epp?estado=`, `GET /epp/{id}` (sst; trabajador solo si es suya)

\- `POST /epp/{id}/aprobar`, `/espera`, `/rechazar`, `/entregar` (sst)



\*\*Inventario\*\* (sst)

\- `GET /recursos?categoria=`, `POST /recursos`, `PATCH /recursos/{id}` (mínimo)

\- `POST /recursos/{id}/movimientos` (ingreso o salida)



\*\*Hallazgos\*\* (sst)

\- `GET /hallazgos?estado=\&riesgo=\&ubicacion=\&responsable=\&q=`, `POST /hallazgos`, `GET /hallazgos/{id}`

\- `POST /hallazgos/{id}/descartar`, `POST /hallazgos/{id}/cerrar`

\- `GET /hallazgos/export.xlsx` y `GET /hallazgos/export.pdf` con los mismos filtros



\*\*Tareas\*\*

\- `POST /hallazgos/{id}/tareas` (sst)

\- `GET /tareas?grupo=` (sst ve todas; áreas solo las de su puesto), `GET /tareas/{id}`

\- `POST /tareas/{id}/aceptar`, `POST /tareas/{id}/modificacion` (área responsable)

\- `POST /tareas/{id}/modificacion/aprobar`, `/rechazar` (sst)

\- `POST /tareas/{id}/pasos/{orden}/completar`, `/reabrir`, `POST /tareas/{id}/terminar` (área responsable)

\- `POST /tareas/{id}/cerrar`, `POST /tareas/{id}/devolver` (sst)



\*\*Resto\*\*

\- `GET/POST/PATCH /tareas-internas` (sst)

\- `GET /biblioteca`, `GET /biblioteca/{id}` (todos); `POST`, `PUT`, `DELETE` (sst)

\- `GET /puestos`, `GET /usuarios/pendientes`, `POST /puestos/{id}/asignar`, `GET /puestos/{id}/historial` (sst)

\- `GET /notificaciones`, `POST /notificaciones/{id}/leer`, `POST /notificaciones/leer-todas` (todos)

\- `GET /dashboard` (sst): indicadores, pendientes, hallazgos por área responsable, tipo, riesgo, ubicación, mes y tiempo promedio de atención



\---



\## Fases y prompts para Claude Code



Cada fase es una sesión. Antes de empezar, usa el modo plan (Shift+Tab) para que Claude Code proponga cómo lo hará; corrige el plan y recién ahí deja que escriba código. Al terminar, `/clear` y sigue con la siguiente.



\### Fase 0: preparar el repositorio



Antes: crea la carpeta del proyecto, copia `CLAUDE.md` a la raíz, este archivo a `docs/PLAN.md` y el prototipo a `docs/mvp-referencia.html`.



> Lee CLAUDE.md y docs/PLAN.md. Crea la estructura del monorepo descrita en CLAUDE.md: módulo Go en /backend con un `main.go` que levante chi en :8080 con `GET /api/health`; proyecto Vite + React + TypeScript + Tailwind en /frontend con proxy de /api a :8080; docker-compose.yml con Postgres 16 y un volumen; Makefile con los comandos de CLAUDE.md; .env.example; .gitignore. No implementes funcionalidades todavía.



\*\*Listo cuando\*\* `make db-up`, `make api` y `make web` funcionan y `/api/health` responde.



\### Fase 1: base de datos



> Con golang-migrate, crea la migración inicial a partir del modelo de datos de docs/PLAN.md. Agrega un comando `make seed` que cargue: las áreas (ubicaciones: Almacén A, Almacén B, Zona de carga, Patio de maniobras, Oficinas; áreas responsables: Mantenimiento, Administración, SST, Calidad), los tipos de hallazgo, las categorías de recurso, y los datos de demostración del prototipo docs/mvp-referencia.html (puestos, personas con DNI, hallazgos con su área responsable, tareas, inventario, solicitudes de EPP y biblioteca). Configura sqlc con pgx/v5 y escribe las consultas base de cada tabla en /backend/queries.



\*\*Listo cuando\*\* `make migrate \&\& make seed \&\& make sqlc` corren limpio y puedes consultar los datos en psql.



\### Fase 2: núcleo del backend y autenticación



> Implementa el núcleo del backend: configuración por variables de entorno, pool de pgx, helper de transacciones, formato de errores JSON de CLAUDE.md, middleware de logging y recuperación. Autenticación: registro con DNI (estado pendiente), login con DNI y contraseña (bcrypt), JWT en cookie httpOnly, `GET /me` que devuelva usuario, puesto activo y rol, middleware `RequireRole`. Un usuario pendiente no puede entrar y recibe el mensaje "Tu registro está pendiente: SST aún no te asigna un puesto." Implementa también la subida de archivos a ./data/uploads con validación de tipo y tamaño, y un servicio `Historial` y otro `Notificar` que reciban la transacción. Escribe pruebas para login, registro y permisos por rol.



\*\*Listo cuando\*\* las pruebas pasan y el login de cada usuario de demostración devuelve su rol.



\### Fase 3: base del frontend



> Construye la base del frontend: cliente HTTP con manejo de errores de la API, TanStack Query, React Router con rutas protegidas por rol, pantalla de login y registro con DNI, y un layout por rol con la barra lateral del prototipo (nombre, puesto, DNI, menú con contadores). Traslada los colores, la tipografía (Barlow y Barlow Condensed) y los componentes visuales del prototipo a Tailwind: Chip de estado, Banner, Modal, Tabs, Card, PhotoField (con cámara en el celular) y Toast. La vista del trabajador debe verse bien en 375 px de ancho.



\*\*Listo cuando\*\* cada uno de los cuatro usuarios de demostración entra y ve su menú.



\### Fase 4: ciclo de EPP e inventario



> Implementa de punta a punta el ciclo de EPP e inventario según CLAUDE.md. Backend: endpoints de EPP e inventario, la máquina de estados de la solicitud con pruebas de cada transición, y la entrega en una sola transacción que descuenta stock, registra el movimiento, escribe historial y notifica al trabajador (y a SST si el stock queda en o bajo el mínimo). Al registrar un ingreso de un recurso con solicitudes en espera, notifica a SST. Frontend: para el trabajador, Solicitar EPP, Mis solicitudes con la línea de estado y el detalle; para SST, Solicitudes de EPP con pestañas por estado, el detalle con sus acciones, e Inventario con pestañas por categoría, ingreso, salida y mínimo. Usa el prototipo como referencia de textos y estados.



\*\*Listo cuando\*\* un trabajador pide un casco con foto, SST lo aprueba y lo entrega, el stock baja en 1 y el trabajador ve "Entregada".



\### Fase 5: ciclo de hallazgo y tareas



> Implementa de punta a punta hallazgos y tareas según CLAUDE.md. Backend: hallazgos con área responsable, tareas con pasos, solicitudes de modificación, validación y devolución; las máquinas de estado del hallazgo y de la tarea con pruebas, incluidas las transiciones prohibidas; regla de cierre: un hallazgo con tareas solo se cierra si todas están cerradas, y uno cuya área responsable es SST o Calidad puede cerrarse sin tareas. Un área solo puede ver y modificar las tareas de su puesto. Frontend: para SST, lista de hallazgos con filtros (estado, riesgo, ubicación, área responsable, búsqueda), registro de hallazgo, detalle con tareas e historial, creación de tarea con pasos, y la lista y el detalle de tareas con aprobar o rechazar modificación, cerrar o devolver con selección de pasos a rehacer. Para Mantenimiento y Administración, el tablero Por hacer, En proceso, Terminadas y el detalle de la tarea con aceptar, pedir modificación, completar paso con foto y comentario, reabrir y terminar.



\*\*Listo cuando\*\* recorres el ciclo completo del prototipo: SST crea hallazgo y tarea, Mantenimiento la completa, SST la devuelve, Mantenimiento la corrige, SST la cierra y luego cierra el hallazgo.



\### Fase 6: biblioteca, tareas internas, puestos y notificaciones



> Implementa: biblioteca (SST publica, edita y elimina con imagen opcional; el trabajador solo lee y filtra por tipo; al publicar se notifica a los trabajadores), tareas internas de SST con las tres columnas, puestos y usuarios (lista de registros pendientes, asignar persona a un puesto cerrando la asignación anterior, historial del puesto) y notificaciones (lista, marcar como leída, marcar todas, contador en el menú, y navegación a la entidad al tocarla). Al reasignar un puesto, la persona nueva debe ver las solicitudes, tareas y notificaciones del puesto.



\*\*Listo cuando\*\* registras un DNI nuevo, SST lo asigna al puesto de un operario y esa persona entra viendo el historial del puesto.



\### Fase 7: dashboard y exportación



> Implementa `GET /api/dashboard` con consultas SQL agregadas (no calcules en Go lo que puede hacer Postgres): hallazgos abiertos, abiertos de riesgo alto, tiempo promedio de atención (de created\_at a cerrado\_at, en días), solicitudes nuevas, tareas por validar, recursos en o bajo el mínimo, la lista de pendientes, y los hallazgos (sin descartados) agrupados por área responsable con cantidad, abiertos y porcentaje, por tipo, por nivel de riesgo, por ubicación, por mes (últimos 6, apilado por riesgo) y el tiempo promedio de atención por riesgo. Frontend: pantalla de dashboard como en el prototipo, con Recharts; al tocar un área responsable, abre la lista de hallazgos filtrada. Exportación de hallazgos a Excel (excelize) y PDF (fpdf) respetando los filtros.



\*\*Listo cuando\*\* los números del dashboard coinciden con una consulta manual en psql y el Excel descargado tiene las mismas filas que la lista filtrada.



\### Fase 8: pruebas de punta a punta y despliegue



> Agrega Playwright con dos pruebas de punta a punta: el ciclo de EPP (trabajador a SST) y el ciclo de hallazgo y tareas (SST a Mantenimiento con devolución). Agrega pruebas de permisos que confirmen que un trabajador no puede ver solicitudes ajenas y que Mantenimiento no puede leer hallazgos, inventario ni tareas de otro puesto. Prepara Dockerfiles de backend y frontend y un docker-compose de producción, y documenta en el README cómo desplegar en un VPS o en Railway o Render.



\*\*Listo cuando\*\* `make test` y Playwright pasan en limpio.



\---



\## Consejos para trabajar con Claude Code



\- Dale siempre el contexto de la fase y pídele que lea `CLAUDE.md` primero; no le pidas varias fases a la vez.

\- Pídele las pruebas de las máquinas de estado antes que la implementación. Son la parte con más riesgo de errores silenciosos.

\- Cuando algo se vea distinto al prototipo, pásale una captura de las dos pantallas y pídele que las iguale.

\- Revisa cada migración antes de aplicarla. Si cambias el esquema después de la fase 1, crea una migración nueva en lugar de editar la anterior.

\- Haz commit al cerrar cada fase para poder volver atrás si una sesión se desvía.

