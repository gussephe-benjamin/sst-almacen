\# SST Almacén



Plataforma de Seguridad y Salud en el Trabajo (SST) para un almacén. Reemplaza el manejo informal (papel, WhatsApp, Excel) por un flujo con trazabilidad: solicitudes de EPP, hallazgos de inspección y tareas con evidencia hasta su cierre.



La referencia visual y funcional es el prototipo navegable en `docs/mvp-referencia.html`. Ábrelo en el navegador antes de construir una pantalla: los textos, estados y flujos de esta versión deben coincidir con él.



\## Roles



| Rol | Quién es | Qué hace |

|---|---|---|

| `trabajador` | Personal operativo y administrativo | Solicita cambio de EPP con foto, ve sus solicitudes, consulta la biblioteca |

| `sst` | Área de SST | Dashboard, hallazgos, tareas a áreas, solicitudes de EPP, inventario, tareas internas, biblioteca, puestos |

| `mantenimiento` | Área de apoyo | Recibe tareas, acepta o pide modificación, sube evidencia por paso |

| `administracion` | Área de apoyo | Igual que mantenimiento |



Los trabajadores NO registran hallazgos ni observaciones. Solo SST registra hallazgos.



\## Reglas de negocio que no se negocian



1\. \*\*Las cuentas son del puesto, no de la persona.\*\* Un puesto tiene un rol. Una persona (identificada por DNI) se asigna a un puesto. Cuando cambia el personal, la nueva persona hereda el puesto y su historial. Los puestos no se eliminan.

2\. \*\*Registro con DNI.\*\* Todo usuario se registra con DNI (8 dígitos), cualquier correo y celular propio. Queda en estado `pendiente` hasta que SST lo asigna a un puesto.

3\. \*\*Trazabilidad.\*\* Toda acción que cambia un estado escribe una fila en `historial` con fecha, usuario, puesto y acción, dentro de la misma transacción.

4\. \*\*Privacidad.\*\* Un trabajador solo ve sus propias solicitudes. Un área de apoyo solo ve las tareas asignadas a su puesto (nunca inventario, hallazgos completos ni tareas internas de SST). Esto se valida en el backend, no solo en la UI.

5\. \*\*Entregar un EPP descuenta stock\*\* en la misma transacción. Si el stock queda en o bajo el mínimo, se notifica a SST.

6\. \*\*Cada hallazgo tiene un área responsable de solucionarlo\*\* (Mantenimiento, Administración, SST, Calidad). Mantenimiento y Administración lo resuelven mediante tareas. SST y Calidad pueden cerrarlo directamente.

7\. \*\*Un hallazgo con tareas solo se cierra\*\* cuando todas sus tareas están `cerrada`.



\## Máquinas de estado



Implementarlas en la capa de servicio, con pruebas unitarias para cada transición permitida y para las prohibidas.



\*\*Solicitud de EPP\*\*

\- `enviada` → `aprobada` (requiere stock > 0) | `en\_espera` (requiere fecha estimada) | `rechazada` (requiere motivo)

\- `en\_espera` → `aprobada` | `rechazada`

\- `aprobada` → `entregada` (requiere `epp\_anterior\_devuelto = true` y `formato\_firmado = true`; descuenta 1 de stock)



\*\*Hallazgo\*\*

\- `registrado` → `en\_atencion` (al crear su primera tarea) | `descartado` (requiere motivo; sin tareas activas) | `cerrado` (solo si el área responsable es SST o Calidad)

\- `en\_atencion` → `cerrado` (todas sus tareas cerradas) | `descartado` (sin tareas activas)



\*\*Tarea a un área\*\*

\- `asignada` → `en\_proceso` (el responsable acepta) | `modificacion` (pide cambio de plazo o alcance)

\- `modificacion` → `en\_proceso` (SST aprueba; si es de plazo, actualiza la fecha límite) | `asignada` (SST rechaza con respuesta)

\- `en\_proceso` → `por\_validar` (todos los pasos con foto y comentario)

\- `por\_validar` → `cerrada` (SST valida) | `devuelta` (SST devuelve con comentario y marca qué pasos rehacer)

\- `devuelta` → `por\_validar` (el responsable corrige y reenvía)



\## Stack



\- \*\*Backend:\*\* Go 1.22+, `net/http` con `go-chi/chi/v5`, `jackc/pgx/v5` (pool), `sqlc` para generar el acceso a datos, `golang-migrate` para migraciones, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `xuri/excelize/v2` para Excel, `go-pdf/fpdf` para PDF, `stretchr/testify` para pruebas.

\- \*\*Base de datos:\*\* PostgreSQL 16 en Docker.

\- \*\*Frontend:\*\* Vite, React 18, TypeScript estricto, Tailwind CSS, React Router, TanStack Query, React Hook Form con Zod, `lucide-react` para íconos, Recharts para el dashboard.

\- \*\*Archivos (fotos):\*\* disco local en `./data/uploads` para esta versión, servidos por un endpoint autenticado. Nunca guardar imágenes como `bytea` en la base.



\## Estructura



```

/backend

&#x20; cmd/api/main.go

&#x20; internal/

&#x20;   config/        variables de entorno

&#x20;   db/            código generado por sqlc (no editar a mano)

&#x20;   httpapi/       router, handlers, middleware (auth, rol, errores)

&#x20;   service/       reglas de negocio y máquinas de estado, una carpeta por dominio

&#x20;   storage/       guardado de archivos

&#x20; migrations/      archivos .up.sql y .down.sql

&#x20; queries/         consultas .sql para sqlc

&#x20; sqlc.yaml

/frontend

&#x20; src/

&#x20;   api/           cliente HTTP y tipos de respuesta

&#x20;   features/      una carpeta por dominio (epp, hallazgos, tareas, inventario, ...)

&#x20;   layouts/       layout por rol

&#x20;   components/    componentes compartidos (Chip, Banner, Modal, PhotoField, ...)

&#x20;   routes/

/docs

&#x20; mvp-referencia.html

&#x20; PLAN.md

docker-compose.yml

Makefile

```



\## Convenciones



\- SQL primero: el esquema vive en `migrations/`, las consultas en `queries/`, y `sqlc generate` produce el código. Sin ORM.

\- Handlers delgados: parsean, validan y llaman al servicio. La lógica vive en `service/`.

\- Toda mutación corre en una transacción que incluye la escritura en `historial` y, si corresponde, en `notificaciones`.

\- Errores de la API en JSON: `{"error": {"code": "stock\_insuficiente", "message": "No hay stock para aprobar esta solicitud."}}`. Los mensajes van en español, listos para mostrarse al usuario.

\- Los códigos visibles (`H-001`, `T-001`, `E-001`) se calculan a partir del `id`; no se guardan aparte.

\- Autorización con middleware `RequireRole(...)` más verificación de propiedad dentro del servicio.

\- Textos de la interfaz en español, en el mismo tono que el prototipo.

\- La vista del trabajador se diseña primero para celular (RNF-02); SST y áreas, para laptop, pero deben funcionar en celular.



\## Comandos



```

make db-up        # levanta Postgres con Docker

make migrate      # aplica migraciones

make seed         # carga catálogos y datos de demostración

make sqlc         # regenera internal/db

make api          # corre el backend en :8081

make web          # corre el frontend en :5173

make test         # go test ./... y npm test

```



\## Cómo trabajar en este repo



\- Trabaja una fase de `docs/PLAN.md` a la vez. Antes de escribir código, propone un plan corto y espera aprobación.

\- Escribe primero las pruebas de las máquinas de estado y de los permisos.

\- Al terminar una fase: corre `make test`, revisa que el flujo funcione contra el prototipo y haz un commit con el nombre de la fase.

\- Si una decisión cambia una regla de este archivo, actualízalo en el mismo commit.


## Decisiones de la versión actual

- **Sin sqlc por ahora.** `sqlc` necesita CGO en Windows. Las consultas viven en `internal/service` y usan pgx directo; las lecturas devuelven JSON generado por Postgres (`json_agg` / `row_to_json`). `internal/db` y `queries/` no se usan todavía.
- **Servicio en un solo paquete** `internal/service` (un archivo por dominio), en lugar de una carpeta por dominio.
- **Migraciones y seed** se ejecutan con el mismo binario: `go run ./cmd/api migrate` y `go run ./cmd/api seed`. El seed reinicia todas las tablas.
- **Puertos:** API en `:8081` (el `:8080` estaba ocupado en la máquina de desarrollo), Postgres en `:5433`.
- **Frontend simple:** React 19, Tailwind 4, sin React Hook Form/Zod ni Recharts (formularios controlados y barras en CSS). Se pueden incorporar después.
- Pendiente: pruebas de integración de permisos en Go, Playwright y Dockerfiles de producción (fase 8).
