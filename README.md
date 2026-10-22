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

Abre http://localhost:5173. Usuarios de demostración (contraseña `demo1234`):

| Rol | Persona | DNI |
|---|---|---|
| Trabajador | Luis Quispe | 71234567 |
| Trabajador | Pedro Huamán | 70456123 |
| SST | Ana Torres | 45678912 |
| Mantenimiento | Carlos Mendoza | 43219876 |
| Administración | Rosa Chávez | 46781234 |
| Pendiente de puesto | Kevin Salazar | 74561230 |

Variables de entorno: ver `.env.example`.
