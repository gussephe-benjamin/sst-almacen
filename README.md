# SST Almacén

Plataforma de SST para almacén: solicitudes de EPP, inventario, hallazgos, tareas a áreas con evidencia, biblioteca, puestos y dashboard.

## Requisitos

Go 1.22+, Node 20+, Docker.

## Arranque

```
make db-up      # Postgres 16 en Docker (puerto 5433)
make migrate    # crea el esquema
make seed       # reinicia la base y carga los datos de demostración
make api        # backend en :8081
make web        # frontend en :5173 (proxy de /api a :8081)
make test       # pruebas de las máquinas de estado
```

Abre http://localhost:5173. Usuarios de demostración (contraseña `demo1234`, salvo el admin):

| Rol | Persona | DNI |
|---|---|---|
| Administrador | Administrador del sistema | 10000001 (contraseña `admin1234`) |
| Trabajador | Luis Quispe | 71234567 |
| Trabajador | Pedro Huamán | 70456123 |
| SST | Ana Torres | 45678912 |
| Mantenimiento | Carlos Mendoza | 43219876 |
| Administración | Rosa Chávez | 46781234 |
| Pendiente de puesto | Kevin Salazar | 74561230 |

## Cuentas en el `.env`

Copia `.env.example` a `.env`. Las variables `ACCOUNTS_ADMIN`, `ACCOUNTS_SST`, `ACCOUNTS_TRABAJADOR`, `ACCOUNTS_MANTENIMIENTO` y `ACCOUNTS_ADMINISTRACION` definen las cuentas que deben existir siempre, con el formato `DNI|contraseña|Nombres|Puesto|Área|Correo|Celular` (varias cuentas separadas por `;`).

- Se crean al iniciar la API y después de `make seed`, solo si el DNI aún no existe. Si ya existe, no se modifica.
- Las cuentas de **administrador** solo se crean desde el `.env`. El administrador gestiona desde la app (menú *Cuentas y puestos*) las cuentas de Trabajador, SST, Mantenimiento y Administración: crear cuentas y puestos, asignar registros pendientes, cambiar al personal de un puesto, liberar puestos y restablecer contraseñas.
