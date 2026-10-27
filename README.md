# SST Almacén

Plataforma de Seguridad y Salud en el Trabajo para un almacén: solicitudes de EPP con foto, inventario con alertas de stock, hallazgos de inspección, tareas a Mantenimiento y Administración con evidencia por paso, biblioteca, notificaciones y dashboard.

Guía de uso por rol: [USO.md](USO.md).

## Requisitos

- Go 1.22+
- Node 20+
- Docker (para PostgreSQL 16)
- `make` (opcional; cada comando también se puede correr a mano)

## Puesta en marcha (uso real)

1. **Configura el `.env`** a partir de la plantilla:
   ```
   cp .env.example .env
   ```
   - `JWT_SECRET`: valor aleatorio de al menos 32 caracteres.
   - `ACCOUNTS_ADMIN`: viene con el DNI del administrador inicial (`10000001`); completa su contraseña (10 caracteres o más).
   - Deja `APP_ENV=production`. Si publicas la app por HTTPS, usa `COOKIE_SECURE=true`.
2. **Levanta la base y prepárala:**
   ```
   make db-up
   make migrate
   ```
   `migrate` crea las tablas y los catálogos base: áreas, tipos de hallazgo y categorías de inventario.
3. **Compila y arranca:**
   ```
   make build
   make start
   ```
   La aplicación queda en `http://<servidor>:8081`. Al iniciar, la API crea las cuentas del `.env` que aún no existan.
4. **Primer ingreso:** entra con el DNI del administrador. En **Cuentas y puestos**, crea las cuentas del personal (SST, Mantenimiento, Administración y trabajadores). También puedes dejar que el personal se registre con su DNI y asignarlo después.
5. **SST registra el inventario** (EPP, extintores, señalización, etc.) para que los trabajadores puedan pedir EPP.

> Con `APP_ENV=production` la API no inicia si el `JWT_SECRET` es débil o si no hay un administrador con una contraseña segura.

### Reiniciar todo

```
make reset
```
Borra **todos** los datos y fotos, y deja la base como recién instalada (catálogos + cuentas del `.env`). No se puede deshacer.

## Administrador inicial

El administrador inicial entra con el DNI **`10000001`**. Su contraseña no está en el repositorio: se define en el `.env` local (`ACCOUNTS_ADMIN`) y se puede cambiar desde la app con *Cambiar contraseña*.

## Cuentas en el `.env`

Las variables `ACCOUNTS_ADMIN`, `ACCOUNTS_SST`, `ACCOUNTS_TRABAJADOR`, `ACCOUNTS_MANTENIMIENTO` y `ACCOUNTS_ADMINISTRACION` definen cuentas que deben existir siempre. Formato (varias cuentas separadas por `;`):

```
DNI|contraseña|Nombres|Puesto|Área|Correo|Celular
```

- Se crean al iniciar la API solo si el DNI aún no existe. Si ya existe, no se modifica (la contraseña puede haberse cambiado desde la app).
- Las cuentas de **administrador** solo se crean así. El administrador gestiona las demás desde la app.

## Desarrollo

```
make db-up && make migrate
make demo   # BORRA todo y carga datos de demostración (contraseña demo1234)
make api    # backend en :8081
make web    # frontend con recarga en :5173
make test
```

`make demo` se niega a correr con `APP_ENV=production`; para desarrollo cambia el `.env` a `APP_ENV=development`.

## Seguridad incluida

- Sesión en cookie `httpOnly` de 7 días; opción `Secure` para HTTPS.
- Bloqueo temporal tras 5 intentos fallidos por DNI (30 por IP) durante 15 minutos.
- Contraseñas con bcrypt, de 8 caracteres como mínimo (10 para administradores).
- Permisos validados en el servidor: un trabajador solo ve sus solicitudes, cada área solo sus tareas, y solo SST ve el inventario y los hallazgos.
- Las fotos se guardan en disco (`UPLOAD_DIR`) y se sirven solo a quien puede ver la entidad dueña de la foto.

## Respaldo

- **Base de datos:** `docker exec sst-project-db-1 pg_dump -U sst sst > respaldo.sql`
- **Fotos:** copia la carpeta `backend/data/uploads`.
