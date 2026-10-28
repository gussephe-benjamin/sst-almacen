# Guía de uso de SST Almacén

SST Almacén reemplaza el papel, WhatsApp y Excel por un solo lugar donde queda registrado **quién hizo qué y cuándo**. Esto vale para las solicitudes de EPP, los hallazgos de inspección y las tareas que se derivan de ellos.

## Ideas clave

- **La cuenta es del puesto, no de la persona.** Si cambia el operario de un puesto, la persona nueva entra con su propio DNI a la misma cuenta y ve todo el historial del puesto: solicitudes, tareas y avisos. La persona anterior ya no puede entrar.
- **Todo deja rastro.** Cada cambio de estado guarda la fecha, la persona y el puesto en el *Historial*, visible en el detalle de cada registro.
- **Cada quien ve solo lo suyo.** Un trabajador solo ve sus solicitudes. Mantenimiento y Administración solo ven las tareas de su puesto. El inventario y los hallazgos son de SST.
- **Funciona en el celular.** La vista del trabajador está pensada primero para el celular. Las de SST y las áreas, para laptop, pero también se usan desde el teléfono. Al adjuntar una foto desde el celular se abre la cámara.

## Roles

| Rol | Qué hace |
|---|---|
| **Administrador** | Crea y gestiona las cuentas y los puestos de todo el personal. |
| **SST** | Dashboard, hallazgos, tareas a las áreas, solicitudes de EPP, inventario, tareas internas, biblioteca. |
| **Mantenimiento / Administración** | Reciben tareas, las aceptan o piden cambios, y suben una foto y un comentario por cada paso. |
| **Trabajador** | Pide cambio de EPP con foto, sigue sus solicitudes y consulta la biblioteca. |

## Ingreso

- Se ingresa con el **DNI (8 dígitos) y la contraseña**.
- **¿Primera vez?** Usa *Regístrate con tu DNI*. El registro queda **pendiente** hasta que el administrador (o SST) te asigna a tu puesto; mientras tanto, al intentar entrar verás *"Tu registro está pendiente"*.
- **¿Olvidaste tu contraseña?** Pide al administrador que la restablezca. Luego cámbiala desde **Cambiar contraseña** en el menú.
- La sesión dura 7 días en el mismo dispositivo. **Salir** la cierra de inmediato.
- Tras 5 intentos fallidos con el mismo DNI, el ingreso se bloquea 15 minutos.

## Administrador: Cuentas y puestos

1. **Crear cuenta:** elige el rol y luego un puesto existente o *Crear un puesto nuevo* (nombre y área). Completa el DNI, los nombres, el correo y el celular. La contraseña inicial se genera sola (puedes cambiarla). Al terminar, la app muestra las **credenciales para entregar**, con un botón para copiarlas; no se vuelven a mostrar.
2. **Nuevo puesto:** crea un puesto con su nombre, rol y área. Puede quedar **vacante** o asignarse en el mismo paso a una persona sin puesto.
3. **Personas sin puesto:** aparecen los registros nuevos y las personas que salieron de un puesto.
   - *Asignar a un puesto* las activa en un puesto existente o en uno nuevo (*+ Crear un puesto nuevo*). Entran con la contraseña que eligieron al registrarse.
   - *Eliminar* las quita de la lista y les quita el acceso (pide confirmación). Lo que hicieron antes queda en el historial. Si vuelven a registrarse con su DNI, llegan como registro nuevo.
4. **Cambiar personal:** crea la cuenta de la persona nueva en ese puesto. La anterior sale del puesto y queda sin acceso; el historial se conserva.
5. **Liberar:** retira a la persona y deja el puesto vacante (pide confirmación).
6. **Perfil:** se abre tocando el nombre de la persona o el botón *Perfil*, tanto en los puestos como en *Personas sin puesto*. Muestra:
   - Sus datos: DNI, puesto, correo, celular y área.
   - La **contraseña asignada**, es decir, la que entregó el administrador al crear la cuenta o al cambiarla, mientras la persona no la cambie. Las contraseñas que eligen las personas no se pueden ver.
   - **Cambiar contraseña**, para entregarle una nueva; desde ese momento se ve en el perfil.

   Ver o cambiar una contraseña queda registrado en el historial del puesto.
7. **Historial:** muestra quiénes ocuparon el puesto y qué acciones se hicieron desde él.

Usa las pestañas por rol, el buscador (puesto, persona, DNI o área) y *Solo vacantes* para ubicarte rápido. Las cuentas de administrador se definen únicamente en el archivo `.env` del servidor.

## Trabajador

1. **Solicitar EPP:** elige el EPP, el motivo y describe el problema, y toma una **foto** del EPP dañado. El botón *Enviar solicitud* se activa cuando todo está completo.
2. **Mis solicitudes:** cada solicitud muestra su avance (*Enviada → Aprobada → Entregada*). Si queda *En espera*, ves la fecha estimada; si es *Rechazada*, el motivo.
3. **Aprobada:** acércate a SST con el EPP anterior para recibir el nuevo y firmar el formato.
4. **Biblioteca:** afiches, flyers, procedimientos e instructivos, con filtros por tipo.
5. **Notificaciones:** te avisan de cada cambio en tus solicitudes y de los contenidos nuevos de la biblioteca. Al tocar una, vas directo al registro.

## SST

### Dashboard
Muestra los indicadores principales: hallazgos abiertos, abiertos de riesgo alto, días promedio de atención, solicitudes nuevas, tareas por validar y recursos bajo el mínimo. Debajo están la lista de **pendientes** y los gráficos por área responsable, tipo, riesgo, ubicación y mes. Toca un indicador o un área para ir a la lista filtrada.

### Solicitudes de EPP
- Pestañas por estado: *Nuevas, Aprobadas, En espera, Entregadas, Rechazadas, Todas*.
- **Aprobar** requiere stock. **Poner en espera** pide una fecha estimada. **Rechazar** pide un motivo, que ve el trabajador.
- **Registrar entrega:** marca que el trabajador devolvió el EPP anterior y que firmó el formato físico. El stock baja en 1 automáticamente; si queda en el mínimo o por debajo, llega un aviso.

### Inventario
- Pestañas por categoría y un aviso con los recursos en el mínimo o por debajo.
- **Ingreso**, **Salida** y **Mínimo** por recurso, y **Nuevo recurso** para darlos de alta.
- Si ingresa stock de un recurso con solicitudes en espera, llega un aviso para atenderlas.

### Hallazgos
1. **Registrar hallazgo:** descripción, detalle, tipo, ubicación, **área responsable**, nivel de riesgo (la prioridad se propone sola) y una foto.
2. Según el área responsable:
   - **Mantenimiento o Administración:** se resuelve con **tareas**. *Crear tarea* pide el título, las indicaciones, el responsable, la fecha límite y los pasos. El hallazgo pasa a *En atención*.
   - **SST o Calidad:** se puede cerrar directamente.
3. **Cerrar hallazgo:** aparece cuando todas sus tareas están cerradas.
4. **Descartar:** pide un motivo y solo es posible sin tareas activas.
5. Filtra por estado, riesgo, ubicación, área responsable o texto. **Excel** y **PDF** exportan exactamente la lista filtrada.

### Tareas a áreas
- La pestaña *Requieren tu atención* reúne lo que espera respuesta de SST: tareas por validar y solicitudes de modificación.
- **Modificación solicitada:** apruébala (si es de plazo, la fecha límite se actualiza sola) o recházala con una respuesta.
- **Por validar:** revisa la foto y el comentario de cada paso. Luego **Valida y cierra**, o **Devuelve** con un comentario marcando los pasos que hay que rehacer.
- Las fechas vencidas se muestran en rojo.

### Tareas internas, Biblioteca y Puestos
- **Tareas internas:** tablero *Pendientes / En curso / Completadas* para el trabajo propio de SST.
- **Biblioteca:** *Publicar* un contenido (con imagen opcional) avisa a todos los trabajadores. También puedes editarlo o eliminarlo.
- **Puestos y usuarios:** SST también puede asignar los registros pendientes a un puesto.

## Mantenimiento y Administración

1. **Mis tareas:** tablero *Por hacer / En proceso / Terminadas*, con fecha límite y avance de pasos.
2. En una tarea *Asignada*:
   - **Aceptar tarea**, o
   - **Pedir modificación** de *plazo* (con nueva fecha) o de *alcance*, explicando el motivo.
3. **Completar** cada paso con **foto y comentario**. Un paso completado por error se puede **Reabrir**.
4. **Terminar y enviar a SST** cuando todos los pasos tengan evidencia.
5. Si SST la **devuelve**, verás su comentario y los pasos marcados *Rehacer*. Corrígelos y vuelve a enviarla.

## Recorrido típico

**EPP:** el trabajador pide un casco con foto → SST lo aprueba → el trabajador recibe el aviso → SST registra la entrega (EPP devuelto y formato firmado) → el stock baja en 1 → el trabajador ve *Entregada*.

**Hallazgo:** SST registra el hallazgo → crea una tarea para Mantenimiento → Mantenimiento la acepta y completa los pasos con fotos → SST la devuelve para corregir un paso → Mantenimiento corrige y reenvía → SST valida y cierra la tarea → SST cierra el hallazgo.

## Consejos

- Las ventanas se cierran con **Escape** o tocando fuera de ellas.
- Los botones se bloquean mientras procesan, para evitar envíos dobles.
- Las acciones que no se pueden deshacer (cerrar, validar, liberar) piden confirmación.
- Los contadores del menú y las notificaciones se actualizan solos cada 30 a 60 segundos.
- En las páginas de detalle, **Volver** regresa a la lista.
