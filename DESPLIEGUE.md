# Despliegue en Railway

Guía para publicar SST Almacén en [Railway](https://railway.com) durante un periodo de prueba con usuarios reales.

## Cómo queda armado

Un proyecto de Railway con dos servicios:

| Servicio | Qué es |
|---|---|
| **sst-almacen** | La aplicación. Railway construye la imagen con el `Dockerfile` del repositorio; la API en Go sirve también el frontend. Tiene un **volumen en `/data`** donde se guardan las fotos. |
| **Postgres** | La base de datos administrada por Railway. |

Al iniciar, la aplicación hace tres cosas:
1. Aplica las migraciones pendientes.
2. Crea el administrador definido en `ACCOUNTS_ADMIN`, si aún no existe.
3. Valida la configuración de producción. Si el `JWT_SECRET` o la contraseña del admin son débiles, no arranca.

## Costo

- **Plan Hobby:** unos US$5 al mes, que incluyen US$5 de consumo.
- **Esta aplicación con su base:** consume poco, normalmente entre US$5 y US$10 al mes en total.
- **Límite de gasto:** ponlo en *Workspace → Usage → Usage limits*, por ejemplo US$15, para evitar sorpresas.

## Pasos

### 1. Prepara dos secretos

Genéralos en tu computadora y guárdalos en un lugar seguro. **No** los subas al repositorio.

```bash
python -c "import secrets; print(secrets.token_urlsafe(48))"
```

- **JWT_SECRET:** el resultado del comando.
- **Contraseña del admin:** puedes correr el mismo comando, o elegir una de 10 caracteres o más.

### 2. Crea el proyecto

1. Entra a railway.com con tu cuenta de GitHub.
2. Ve a **New Project → Deploy from GitHub repo**. Si Railway lo pide, dale acceso al repositorio privado `sst-almacen`.
3. Elige el repositorio. Railway detecta el `Dockerfile` y `railway.json`. Este primer despliegue va a fallar porque aún faltan las variables; es normal.

### 3. Agrega la base de datos

En el proyecto, ve a **+ Create → Database → PostgreSQL**.

### 4. Variables de la aplicación

En el servicio de la aplicación, ve a **Variables** y agrega:

| Variable | Valor |
|---|---|
| `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` (referencia a la base; Railway la completa) |
| `JWT_SECRET` | el secreto del paso 1 |
| `ACCOUNTS_ADMIN` | `10000001\|TU-CONTRASEÑA\|Administrador del sistema\|Administrador del sistema\|Dirección` |
| `COOKIE_SECURE` | `true` |
| `TRUST_PROXY` | `true` |
| `PORT` | `8080` |

En `ACCOUNTS_ADMIN`, escribe las barras `|` sin la barra invertida `\`, que en esta tabla solo sirve para mostrarlas. Puedes usar tu DNI en lugar de `10000001`.

El `Dockerfile` ya define `APP_ENV=production`, `STATIC_DIR`, `UPLOAD_DIR=/data/uploads` y `TZ=America/Lima`.

### 5. Volumen para las fotos

1. Haz clic derecho sobre el servicio de la aplicación y elige **Attach volume**.
2. Usa la ruta de montaje **`/data`**.

Sin volumen, las fotos se borran en cada despliegue.

### 6. Dominio

1. En el servicio de la aplicación, ve a **Settings → Networking → Generate Domain**.
2. Usa el puerto **8080**.

Obtienes una dirección como `sst-almacen-production.up.railway.app`, con HTTPS incluido. Si tienes un dominio propio, agrégalo en **Custom Domain**.

### 7. Despliega y verifica

1. Railway vuelve a desplegar al guardar las variables. Si no lo hace, usa **Deploy** en el servicio.
2. En **Deployments → View logs** deben aparecer estas líneas:
   ```
   migraciones aplicadas
   cuenta creada desde .env: 10000001 (admin)
   API en :8080 (production)
   ```
3. Abre el dominio y entra con el DNI y la contraseña del admin.
4. Primeros pasos en la app:
   - El admin crea las cuentas, en *Cuentas y puestos*.
   - SST registra el stock de los EPP con *Inventario → Ingreso*.

## Actualizaciones

Cada `git push` a `main` vuelve a desplegar la aplicación automáticamente. Las migraciones nuevas se aplican solas al iniciar. Los datos y las fotos se conservan.

## Respaldos

Con personas reales, haz respaldos al menos una vez por semana.

- **Base de datos:**
  - En el servicio Postgres, revisa la pestaña **Backups**: Railway ofrece respaldos programados según el plan.
  - Respaldo manual desde tu computadora: copia `DATABASE_PUBLIC_URL` (Postgres → Variables) y ejecuta este comando con Docker:
    ```bash
    docker run --rm postgres:16 pg_dump "PEGA_AQUI_DATABASE_PUBLIC_URL" > respaldo.sql
    ```
- **Fotos:** el volumen también tiene la opción **Backups** en Railway.

## Si algo falla

| Síntoma en los logs | Causa |
|---|---|
| `configuración insegura: JWT_SECRET…` | `JWT_SECRET` falta o es corto. |
| `configuración insegura: … administrador …` | Falta `ACCOUNTS_ADMIN`, o la contraseña tiene menos de 10 caracteres. |
| Error de conexión a la base | `DATABASE_URL` no apunta a `${{Postgres.DATABASE_URL}}`. |
| La app abre pero no mantiene la sesión | Entras por `http://`. Usa el dominio con `https://`; `COOKIE_SECURE=true` lo exige. |
| Las fotos desaparecen tras desplegar | Falta el volumen en `/data`. |

**Cambiar la contraseña del admin:** cambiar `ACCOUNTS_ADMIN` después del primer arranque **no** modifica la cuenta, porque solo se crea si no existe. Cámbiala desde la app con *Cambiar contraseña*.
