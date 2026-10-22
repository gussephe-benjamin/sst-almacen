package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"sst-almacen/internal/storage"
	"sst-almacen/migrations"
)

// ago devuelve una expresión SQL para "hace n días" (n puede ser negativo: futuro).
const ago = `now() - ($%d::float * interval '1 day')`

// resetDB borra todos los datos y archivos subidos y vuelve a cargar los catálogos base.
func resetDB(ctx context.Context, tx pgx.Tx, files *storage.Store) error {
	var tables []string
	rows, err := tx.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename <> 'schema_migrations'`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tables = append(tables, t)
	}
	rows.Close()
	if _, err := tx.Exec(ctx, `TRUNCATE `+strings.Join(tables, ",")+` RESTART IDENTITY CASCADE`); err != nil {
		return err
	}
	catalogos, err := migrations.FS.ReadFile("0003_catalogos.up.sql")
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, string(catalogos)); err != nil {
		return err
	}
	return os.RemoveAll(files.Dir)
}

// seed reinicia la base y carga los datos de demostración del prototipo (solo para desarrollo).
// Todos los usuarios de demostración usan la contraseña "demo1234".
func seed(ctx context.Context, pool *pgxpool.Pool, files *storage.Store) error {
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		ex := func(sql string, args ...any) {
			if _, err := tx.Exec(ctx, sql, args...); err != nil {
				panic(err.Error() + "\n" + sql)
			}
		}
		if err := resetDB(ctx, tx, files); err != nil {
			return err
		}
		// Mismo orden que la migración de catálogos (ids 1..n).
		tipos := []string{"Almacenamiento y estanterías", "Orden y limpieza", "Equipos de emergencia", "Señalización", "Instalaciones eléctricas",
			"Tránsito de montacargas", "Infraestructura", "Uso de EPP", "Otro"}

		// Puestos (ids 1-7)
		for _, p := range [][]any{
			{"Operario de almacén 1", "trabajador", 1}, {"Operario de almacén 2", "trabajador", 2}, {"Auxiliar de despacho", "trabajador", 3},
			{"Operador de montacargas", "trabajador", 4}, {"Jefa de SST", "sst", 8}, {"Técnico de mantenimiento", "mantenimiento", 6},
			{"Asistente de administración", "administracion", 7},
		} {
			ex(`INSERT INTO puestos (nombre, rol, area_id) VALUES ($1,$2,$3)`, p...)
		}

		// Personas (ids 1-7 activos, 8 anterior, 9 pendiente)
		hash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
		personas := [][]any{
			{"71234567", "Luis Quispe", "luisquispe@gmail.com", "987 112 334", 1, "activo", 420},
			{"70456123", "Pedro Huamán", "pedro.hc@hotmail.com", "965 774 120", 2, "activo", 260},
			{"72561894", "María Flores", "mflores.d@gmail.com", "944 208 991", 3, "activo", 190},
			{"70918273", "Jorge Ramos", "jramos88@gmail.com", "912 336 450", 4, "activo", 610},
			{"45678912", "Ana Torres", "atorres.sst@gmail.com", "998 450 216", 8, "activo", 730},
			{"43219876", "Carlos Mendoza", "cmendoza.mant@gmail.com", "976 551 803", 6, "activo", 540},
			{"46781234", "Rosa Chávez", "rchavez.adm@gmail.com", "953 667 142", 7, "activo", 320},
			{"70112908", "Raúl Paredes", "raul.paredes@gmail.com", "955 100 200", 2, "inactivo", 900},
			{"74561230", "Kevin Salazar", "kevin.salazar@gmail.com", "981 640 327", 2, "pendiente", 1},
		}
		for _, p := range personas {
			ex(`INSERT INTO usuarios (dni, nombres, correo, celular, password_hash, area_id, estado, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,`+sprintf(ago, 8)+`)`,
				p[0], p[1], p[2], p[3], string(hash), p[4], p[5], p[6])
		}
		for i := 1; i <= 7; i++ {
			ex(`INSERT INTO asignaciones_puesto (puesto_id, usuario_id, desde, asignado_por) VALUES ($1,$1,`+sprintf(ago, 2)+`,5)`, i, personas[i-1][6])
		}
		ex(`INSERT INTO asignaciones_puesto (puesto_id, usuario_id, desde, hasta, asignado_por) VALUES (2, 8, now() - interval '900 days', now() - interval '261 days', 5)`)

		// Foto de muestra
		img := image.NewRGBA(image.Rect(0, 0, 320, 200))
		for x := 0; x < 320; x++ {
			for y := 0; y < 200; y++ {
				img.Set(x, y, color.RGBA{uint8(60 + x/4), uint8(90 + y/4), 120, 255})
			}
		}
		var buf bytes.Buffer
		png.Encode(&buf, img)
		name, size, err := files.Save(&buf, ".png")
		if err != nil {
			return err
		}
		var foto string
		tx.QueryRow(ctx, `INSERT INTO archivos (ruta, mime, bytes, subido_por) VALUES ($1,'image/png',$2,5) RETURNING id::text`, name, size).Scan(&foto)

		// Inventario
		inv := [][]any{
			{1, "Casco de seguridad", 12, 5}, {1, "Guantes anticorte (par)", 30, 10}, {1, "Lentes de seguridad", 4, 6},
			{1, "Zapatos de seguridad talla 42", 0, 3}, {1, "Zapatos de seguridad talla 40", 5, 3}, {1, "Chaleco reflectivo", 15, 5},
			{1, "Tapones auditivos (par)", 50, 20}, {1, "Arnés de seguridad", 3, 2},
			{2, "Extintor PQS 6 kg", 8, 6}, {2, "Extintor CO2 5 kg", 2, 2},
			{3, "Señal de salida", 10, 4}, {3, "Señal de riesgo eléctrico", 3, 3}, {3, "Señal de ruta de evacuación", 14, 8},
			{4, "Botiquín completo", 4, 3}, {4, "Paquete de gasas estériles", 6, 10}, {5, "Luz de emergencia LED", 5, 4},
		}
		for _, r := range inv {
			ex(`INSERT INTO recursos (categoria_id, nombre, stock, stock_minimo) VALUES ($1,$2,$3,$4)`, r...)
		}

		// Hallazgos: desc, detalle, tipo, ubicación, área resp, riesgo, prioridad, estado, días, cerrado (días o -1)
		ubic := map[string]int{"Almacén A": 1, "Almacén B": 2, "Zona de carga": 3, "Patio de maniobras": 4, "Oficinas": 5}
		resp := map[string]int{"Mantenimiento": 6, "Administración": 7, "SST": 8, "Calidad": 9}
		tipoID := func(n string) int {
			for i, t := range tipos {
				if t == n {
					return i + 1
				}
			}
			return len(tipos)
		}
		hs := [][]any{
			{"Travesaño del rack 4 deformado", "El travesaño inferior del rack 4 presenta una deformación visible por impacto de montacargas.", "Almacenamiento y estanterías", "Almacén A", "Mantenimiento", "alto", "alta", "cerrado", 150.0, 143.0},
			{"Pasadizo obstruido con parihuelas", "Parihuelas vacías apiladas en el pasadizo principal reducen el ancho libre de evacuación.", "Orden y limpieza", "Almacén B", "SST", "medio", "media", "cerrado", 131.0, 129.0},
			{"Extintor PQS con carga vencida", "Extintor junto a la puerta de la zona de carga con tarjeta de recarga vencida hace dos meses.", "Equipos de emergencia", "Zona de carga", "Administración", "alto", "alta", "cerrado", 112.0, 105.0},
			{"Señal de salida despintada", "La señal de salida del lado norte está decolorada y no es legible a más de 5 metros.", "Señalización", "Almacén A", "SST", "bajo", "baja", "cerrado", 96.0, 82.0},
			{"Cable expuesto en tablero eléctrico", "Tablero del patio con la puerta abierta y un conductor sin aislamiento.", "Instalaciones eléctricas", "Patio de maniobras", "Mantenimiento", "alto", "alta", "cerrado", 81.0, 77.0},
			{"Derrame de líquido en pasillo 2", "Derrame reportado en el pasillo 2.", "Orden y limpieza", "Almacén A", "SST", "bajo", "baja", "descartado", 70.0, -1.0},
			{"Montacargas sin alarma de retroceso", "El montacargas 02 no emite alarma sonora al retroceder.", "Tránsito de montacargas", "Patio de maniobras", "Mantenimiento", "alto", "alta", "cerrado", 61.0, 52.0},
			{"Botiquín incompleto en oficina de despacho", "Faltan gasas, vendas y alcohol en el botiquín de la oficina de despacho.", "Equipos de emergencia", "Oficinas", "Administración", "medio", "media", "cerrado", 46.0, 41.0},
			{"Luz de emergencia del pasillo 3 no enciende", "En la prueba de corte de energía, la luz de emergencia del pasillo 3 no encendió.", "Equipos de emergencia", "Almacén A", "Mantenimiento", "alto", "alta", "en_atencion", 3.0, -1.0},
			{"Fisuras en el piso de la zona de carga", "Fisuras de 1 a 2 cm en la losa, en la ruta de tránsito de transpaletas.", "Infraestructura", "Zona de carga", "Mantenimiento", "medio", "media", "en_atencion", 9.0, -1.0},
			{"Rutas de evacuación sin señalizar", "Faltan señales de ruta de evacuación en el almacén B tras la redistribución de racks.", "Señalización", "Almacén B", "Administración", "medio", "media", "en_atencion", 13.0, -1.0},
			{"Extintores CO2 por recargar", "Los dos extintores de CO2 del área eléctrica vencen su recarga este mes.", "Equipos de emergencia", "Zona de carga", "Administración", "alto", "alta", "en_atencion", 6.0, -1.0},
			{"Cajas apiladas sobre la altura permitida", "Columna de cajas en el almacén B supera la línea de altura máxima de apilamiento.", "Almacenamiento y estanterías", "Almacén B", "Calidad", "medio", "media", "registrado", 1.0, -1.0},
			{"Mancha de aceite junto al montacargas 01", "Mancha de aceite en el área de estacionamiento del montacargas 01.", "Orden y limpieza", "Patio de maniobras", "SST", "bajo", "baja", "registrado", 0.2, -1.0},
		}
		for i, h := range hs {
			id := i + 1
			var cerrado any
			if h[9].(float64) >= 0 {
				cerrado = h[9]
			}
			motivo := any(nil)
			if h[7] == "descartado" {
				motivo = "Al verificar, el derrame ya había sido limpiado por el personal de turno y no se encontró fuente de fuga."
			}
			ex(`INSERT INTO hallazgos (descripcion, detalle, tipo_id, ubicacion_id, area_responsable_id, nivel_riesgo, prioridad, responsable_id, estado, motivo_descarte, created_at, cerrado_at)
				VALUES ($1,$2,$3,$4,$5,$6,$7,5,$8,$9,`+sprintf(ago, 10)+`, CASE WHEN $11::float IS NULL THEN NULL ELSE `+sprintf(ago, 11)+` END)`,
				h[0], h[1], tipoID(h[2].(string)), ubic[h[3].(string)], resp[h[4].(string)], h[5], h[6], h[7], motivo, h[8], cerrado)
			ex(`INSERT INTO hallazgo_fotos (hallazgo_id, archivo_id) VALUES ($1,$2)`, id, foto)
			ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES ('hallazgo',$1,'Registró el hallazgo durante la inspección',5,5,`+sprintf(ago, 2)+`)`, id, h[8])
			if cerrado != nil {
				ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES ('hallazgo',$1,'Cerró el hallazgo',5,5,`+sprintf(ago, 2)+`)`, id, cerrado)
			}
		}

		// Tareas: hallazgo, título, indicaciones, puesto, creada (días), límite (días, negativo = futuro), estado, cerrada (días o -1), pasos
		type paso struct {
			t, com string
			dias   float64
		}
		ts := []struct {
			h       int
			titulo  string
			ind     string
			puesto  int
			creada  float64
			limite  float64
			estado  string
			cerrada float64
			pasos   []paso
		}{
			{9, "Reparar luz de emergencia del pasillo 3", "Revisar y dejar operativa la luz de emergencia del pasillo 3. Si no tiene reparación, reemplazarla por una del inventario.", 6, 2, -3, "asignada", -1,
				[]paso{{"Revisar batería y conexión", "", 0}, {"Reparar o reemplazar el equipo", "", 0}, {"Probar el encendido con corte de energía", "", 0}}},
			{10, "Resanar fisuras del piso en zona de carga", "Resanar las fisuras con mortero epóxico. Mantener la zona señalizada hasta el secado completo.", 6, 8, -1, "en_proceso", -1,
				[]paso{{"Delimitar y señalizar la zona", "Zona delimitada con conos y cinta. Se desvió el tránsito de transpaletas por el lado sur.", 2}, {"Resanar fisuras con mortero epóxico", "", 0}, {"Retirar la señalización tras el secado", "", 0}}},
			{11, "Comprar e instalar señales de ruta de evacuación", "Adquirir 12 señales fotoluminiscentes de ruta de evacuación y coordinar su instalación según el plano actualizado.", 7, 12, -2, "por_validar", -1,
				[]paso{{"Cotizar con dos proveedores", "Se cotizó con dos proveedores. Se eligió el de menor plazo de entrega.", 9}, {"Emitir la orden de compra", "Orden de compra OC-2026-0418 aprobada por gerencia.", 6}, {"Coordinar la instalación", "El proveedor instaló las 12 señales el jueves.", 1}}},
			{12, "Contratar la recarga de extintores CO2", "Contratar el servicio de recarga de los dos extintores de CO2 de 5 kg y gestionar extintores de reemplazo mientras dure el servicio.", 7, 5, -1, "modificacion", -1,
				[]paso{{"Solicitar cotización del servicio", "", 0}, {"Emitir la orden de servicio", "", 0}, {"Recibir los extintores recargados", "", 0}}},
			{8, "Reponer insumos del botiquín de despacho", "Comprar gasas, vendas y alcohol para completar el botiquín.", 7, 45, 40, "cerrada", 41,
				[]paso{{"Comprar insumos", "Compra en farmacia con caja chica.", 43}, {"Reponer el botiquín", "Botiquín completo según lista de verificación.", 42}}},
			{7, "Instalar alarma de retroceso en montacargas 02", "Instalar alarma sonora de retroceso y probarla.", 6, 60, 52, "cerrada", 52,
				[]paso{{"Instalar la alarma", "Alarma de 97 dB instalada.", 54}, {"Probar en retroceso", "Prueba conforme con el operador.", 53}}},
			{5, "Aislar conductor y cerrar tablero del patio", "Aislar el conductor expuesto y colocar candado en el tablero.", 6, 80, 77, "cerrada", 77,
				[]paso{{"Aislar el conductor", "Conductor reemplazado.", 78}, {"Colocar candado", "Candado instalado; llave en mantenimiento.", 78}}},
		}
		for i, t := range ts {
			id := i + 1
			area := 6
			if t.puesto == 7 {
				area = 7
			}
			var cerrada any
			if t.cerrada >= 0 {
				cerrada = t.cerrada
			}
			ex(`INSERT INTO tareas (hallazgo_id, titulo, indicaciones, area_id, puesto_id, creada_por, fecha_limite, estado, created_at, cerrada_at)
				VALUES ($1,$2,$3,$4,$5,5,(`+sprintf(ago, 6)+`)::date,$7,`+sprintf(ago, 8)+`, CASE WHEN $9::float IS NULL THEN NULL ELSE `+sprintf(ago, 9)+` END)`,
				t.h, t.titulo, t.ind, area, t.puesto, t.limite, t.estado, t.creada, cerrada)
			ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES ('tarea',$1,'Creó la tarea y la asignó',5,5,`+sprintf(ago, 2)+`)`, id, t.creada)
			for j, p := range t.pasos {
				if p.com != "" {
					ex(`INSERT INTO tarea_pasos (tarea_id, orden, titulo, completado, evidencia_id, comentario, completado_por, completado_at)
						VALUES ($1,$2,$3,true,$4,$5,$6,`+sprintf(ago, 7)+`)`, id, j+1, p.t, foto, p.com, t.puesto, p.dias)
					ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES ('tarea',$1,$2,$3,$3,`+sprintf(ago, 4)+`)`,
						id, sprintf("Completó el paso %d con evidencia", j+1), t.puesto, p.dias)
				} else {
					ex(`INSERT INTO tarea_pasos (tarea_id, orden, titulo) VALUES ($1,$2,$3)`, id, j+1, p.t)
				}
			}
		}
		ex(`INSERT INTO solicitudes_modificacion (tarea_id, tipo, nueva_fecha, motivo, solicitada_por, created_at)
			VALUES (4,'plazo',(now() + interval '10 days')::date,'El proveedor habitual tiene carga de CO2 recién para la próxima semana. Otro proveedor cobra el doble.',7, now() - interval '7 hours')`)
		ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES ('tarea',2,'Aceptó la tarea',6,6,now() - interval '7 days'),
			('tarea',3,'Marcó la tarea como terminada',7,7,now() - interval '22 hours'), ('tarea',4,'Solicitó modificación de plazo',7,7,now() - interval '7 hours')`)

		// Solicitudes de EPP: puesto, recurso, motivo, descripción, días, estado
		epp := [][]any{
			{1, 2, "Desgaste por uso", "Los guantes tienen la palma rota y ya no protegen al manipular flejes.", 20.0, "entregada"},
			{1, 4, "Rotura o daño", "La suela del zapato derecho se despegó en la punta.", 4.0, "en_espera"},
			{2, 1, "Rotura o daño", "El casco tiene una fisura en la parte superior después de un golpe con un rack.", 0.15, "enviada"},
			{3, 3, "Desgaste por uso", "Las lunas están rayadas y no se ve bien al leer etiquetas.", 1.0, "enviada"},
			{4, 6, "Desgaste por uso", "El chaleco perdió las cintas reflectivas de la espalda.", 2.0, "aprobada"},
			{2, 7, "Pérdida", "Perdí los tapones en el comedor.", 10.0, "rechazada"},
		}
		for i, e := range epp {
			ex(`INSERT INTO solicitudes_epp (puesto_id, usuario_id, recurso_id, motivo, descripcion, foto_id, estado, created_at) VALUES ($1,$1,$2,$3,$4,$5,$6,`+sprintf(ago, 7)+`)`,
				e[0], e[1], e[2], e[3], foto, e[5], e[4])
			ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES ('solicitud_epp',$1,'Envió la solicitud',$2,$2,`+sprintf(ago, 3)+`)`, i+1, e[0], e[4])
		}
		ex(`UPDATE solicitudes_epp SET epp_anterior_devuelto=true, formato_firmado=true, entregada_at=now() - interval '18 days' WHERE id=1`)
		ex(`UPDATE solicitudes_epp SET fecha_espera=(now() + interval '5 days')::date WHERE id=2`)
		ex(`UPDATE solicitudes_epp SET motivo_rechazo='Los tapones están disponibles en el dispensador del ingreso; no requieren solicitud.' WHERE id=6`)
		ex(`INSERT INTO historial (entidad, entidad_id, accion, usuario_id, puesto_id, created_at) VALUES
			('solicitud_epp',1,'Aprobó la solicitud',5,5,now() - interval '19 days'),
			('solicitud_epp',1,'Marcó como entregada; EPP anterior devuelto y formato firmado',5,5,now() - interval '18 days'),
			('solicitud_epp',2,'Puso la solicitud en espera por falta de stock',5,5,now() - interval '3 days'),
			('solicitud_epp',5,'Aprobó la solicitud',5,5,now() - interval '36 hours'),
			('solicitud_epp',6,'Rechazó la solicitud',5,5,now() - interval '9 days')`)

		// Biblioteca
		for _, b := range [][]any{
			{"afiche", "", "Uso obligatorio de EPP en zona de carga", "Casco, chaleco reflectivo, zapatos de seguridad y guantes en todo momento.", "En la zona de carga circulan montacargas y transpaletas de forma continua. Todo el personal que ingrese debe usar casco, chaleco reflectivo, zapatos con puntera de acero y guantes.\n\nSi tu EPP está dañado, solicita el cambio desde la plataforma.", 40},
			{"procedimiento", "PR-SST-004", "Evacuación ante sismos", "Pasos para evacuar el almacén y llegar al punto de reunión.", "Al iniciar el sismo, aléjate de racks y ventanas y ubícate en una zona segura señalizada.\n\nCuando termine, evacúa por la ruta señalizada sin correr y sin usar el montacargas.\n\nDirígete al punto de reunión del patio de maniobras y repórtate con tu supervisor.", 120},
			{"instructivo", "IN-SST-011", "Uso correcto del extintor PQS", "Cómo usar el extintor en cuatro pasos.", "Retira el pasador de seguridad.\n\nApunta la boquilla a la base del fuego, a unos 2 a 3 metros.\n\nPresiona la manija de forma continua.\n\nBarre de lado a lado hasta apagar el fuego y avisa a SST para su recarga.", 75},
			{"flyer", "", "Levantamiento seguro de cargas", "Dobla las rodillas, no la espalda. Pide ayuda con cargas de más de 25 kg.", "Acércate a la carga y separa los pies al ancho de los hombros.\n\nDobla las rodillas, mantén la espalda recta y levanta con las piernas.\n\nNo gires el tronco con la carga; mueve los pies.\n\nSi la carga supera los 25 kg, pide ayuda o usa una transpaleta.", 30},
			{"procedimiento", "PR-SST-007", "Operación segura de montacargas", "Inspección previa, velocidad máxima y prioridad peatonal.", "Antes de cada turno, completa el check list de inspección del montacargas.\n\nLa velocidad máxima dentro del almacén es de 10 km/h.\n\nEl peatón siempre tiene prioridad; toca la bocina en cruces y esquinas.", 160},
			{"afiche", "", "Mantén los pasadizos libres", "Las rutas de evacuación deben estar despejadas siempre.", "No dejes parihuelas, cajas ni stretch film en los pasadizos marcados con línea amarilla.", 12},
		} {
			ex(`INSERT INTO contenidos_biblioteca (tipo, codigo, titulo, resumen, cuerpo, publicado_por, created_at) VALUES ($1,$2,$3,$4,$5,5,`+sprintf(ago, 6)+`)`, b...)
		}

		// Tareas internas
		for _, t := range [][]any{
			{"Inspección mensual de extintores", -5, "pendiente"}, {"Capacitación sobre uso y cuidado de EPP", -12, "pendiente"},
			{"Actualizar el mapa de riesgos del almacén B", -20, "en_curso"}, {"Revisar la matriz IPERC", -30, "pendiente"},
			{"Simulacro de sismo del segundo semestre", 7, "completada"},
		} {
			ex(`INSERT INTO tareas_internas (titulo, responsable, fecha, estado) VALUES ($1,'Ana Torres',(`+sprintf(ago, 2)+`)::date,$3)`, t...)
		}

		// Notificaciones
		for _, n := range [][]any{
			{"sst", nil, "Pedro Huamán solicitó cambio de Casco de seguridad (E-003).", "epp", 3, 0.15},
			{"sst", nil, "Rosa Chávez solicitó más plazo para T-004.", "tarea", 4, 0.3},
			{"sst", nil, "Rosa Chávez terminó T-003. Revisa la evidencia para cerrarla.", "tarea", 3, 0.9},
			{"sst", nil, "María Flores solicitó cambio de Lentes de seguridad (E-004).", "epp", 4, 1.0},
			{"sst", nil, "Kevin Salazar se registró con DNI y espera asignación de puesto.", "puestos", 0, 1.0},
			{"admin", nil, "Kevin Salazar se registró con DNI y espera asignación de puesto.", "cuentas", 0, 1.0},
			{"sst", nil, "Stock bajo: Lentes de seguridad (4 en stock, mínimo 6).", "inventario", 3, 2.0},
			{"mantenimiento", 6, "Ana Torres te asignó T-001: Reparar luz de emergencia del pasillo 3.", "tarea", 1, 2.0},
			{"trabajador", 1, "Tu solicitud E-002 quedó en espera por falta de stock.", "epp", 2, 3.5},
			{"trabajador", 4, "Tu solicitud E-005 fue aprobada. Acércate a SST para la entrega.", "epp", 5, 1.5},
		} {
			ex(`INSERT INTO notificaciones (para_rol, para_puesto_id, texto, ref_tipo, ref_id, created_at) VALUES ($1,$2,$3,$4,$5,`+sprintf(ago, 6)+`)`, n...)
		}
		return nil
	})
}

var sprintf = fmt.Sprintf
