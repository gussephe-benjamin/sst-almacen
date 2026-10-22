package service

// Máquinas de estado. Solo describen qué transiciones existen; los requisitos
// de cada una (stock, fecha, motivo, pasos completos...) se validan en el servicio.

var transEPP = map[string][]string{
	"enviada":   {"aprobada", "en_espera", "rechazada"},
	"en_espera": {"aprobada", "rechazada"},
	"aprobada":  {"entregada"},
}

var transHallazgo = map[string][]string{
	"registrado":  {"en_atencion", "descartado", "cerrado"},
	"en_atencion": {"cerrado", "descartado"},
}

var transTarea = map[string][]string{
	"asignada":     {"en_proceso", "modificacion"},
	"modificacion": {"en_proceso", "asignada"},
	"en_proceso":   {"por_validar"},
	"por_validar":  {"cerrada", "devuelta"},
	"devuelta":     {"por_validar"},
}

func can(m map[string][]string, from, to string) bool {
	for _, s := range m[from] {
		if s == to {
			return true
		}
	}
	return false
}

func PuedeEPP(from, to string) bool      { return can(transEPP, from, to) }
func PuedeHallazgo(from, to string) bool { return can(transHallazgo, from, to) }
func PuedeTarea(from, to string) bool    { return can(transTarea, from, to) }

func errTrans() *AppError {
	return Conflict("transicion_invalida", "Esta acción no está permitida en el estado actual.")
}
