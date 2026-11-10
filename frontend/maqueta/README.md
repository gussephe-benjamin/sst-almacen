# Maqueta del producto

`docs/maqueta-producto.html` es la plataforma real (los mismos componentes de `src/`) en un solo archivo que se abre con doble clic, sin servidor ni base de datos.

- El botón flotante **Cambiar rol** cambia de rol: Trabajador, SST, Mantenimiento, Administración, Calidad o Administrador. También se puede ingresar desde el login con el DNI de una cuenta de demostración (cualquier contraseña).
- Los datos son de demostración (`fixtures.json`). Los botones responden, pero los cambios no se guardan y no se descargan reportes.
- Para la vista de celular, achica la ventana o usa el modo dispositivo del navegador (F12).

## Regenerarla tras cambios en la interfaz

```bash
npm --prefix frontend run maqueta
```

## Regenerar los datos de demostración

Usa una base aparte para no tocar la de desarrollo. `demo` y `reset` borran la base y la carpeta `UPLOAD_DIR` a la que apuntan.

1. Crea la base: `docker exec sst-project-db-1 psql -U sst -d sst -c "CREATE DATABASE sst_maqueta"`.
2. En una carpeta **sin `.env`**, define `DATABASE_URL` (a `sst_maqueta`), `UPLOAD_DIR` (una carpeta temporal), `PORT=8091` y `JWT_SECRET`. Define también `ACCOUNTS_ADMIN` y `ACCOUNTS_CALIDAD` con datos ficticios y la misma contraseña.
3. Ejecuta `api migrate`, luego `api demo --si`, y después `api` para levantar la API en `:8091`.
4. Ejecuta `MAQUETA_PW=<esa contraseña> python frontend/maqueta/capturar.py`. Agrega ejemplos (afiches, extintor, stock bajo, EPP de un área) y guarda `fixtures.json`.
5. Ejecuta `npm --prefix frontend run maqueta` y borra la base `sst_maqueta`.
