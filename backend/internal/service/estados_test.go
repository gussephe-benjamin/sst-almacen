package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type tr struct{ from, to string }

func check(t *testing.T, name string, fn func(string, string) bool, states []string, allowed []tr) {
	ok := map[tr]bool{}
	for _, a := range allowed {
		ok[a] = true
	}
	for _, f := range states {
		for _, to := range states {
			assert.Equal(t, ok[tr{f, to}], fn(f, to), "%s: %s -> %s", name, f, to)
		}
	}
}

func TestEPP(t *testing.T) {
	check(t, "epp", PuedeEPP, []string{"enviada", "aprobada", "en_espera", "rechazada", "entregada"}, []tr{
		{"enviada", "aprobada"}, {"enviada", "en_espera"}, {"enviada", "rechazada"},
		{"en_espera", "aprobada"}, {"en_espera", "rechazada"},
		{"aprobada", "entregada"},
	})
}

func TestHallazgo(t *testing.T) {
	check(t, "hallazgo", PuedeHallazgo, []string{"registrado", "en_atencion", "cerrado", "descartado"}, []tr{
		{"registrado", "en_atencion"}, {"registrado", "descartado"}, {"registrado", "cerrado"},
		{"en_atencion", "cerrado"}, {"en_atencion", "descartado"},
	})
}

func TestTarea(t *testing.T) {
	check(t, "tarea", PuedeTarea, []string{"asignada", "modificacion", "en_proceso", "por_validar", "devuelta", "cerrada"}, []tr{
		{"asignada", "en_proceso"}, {"asignada", "modificacion"},
		{"modificacion", "en_proceso"}, {"modificacion", "asignada"},
		{"en_proceso", "por_validar"},
		{"por_validar", "cerrada"}, {"por_validar", "devuelta"},
		{"devuelta", "por_validar"},
	})
}

func TestPuedeCerrarHallazgo(t *testing.T) {
	assert.True(t, PuedeCerrarHallazgo("registrado", "SST", 0, 0))
	assert.True(t, PuedeCerrarHallazgo("registrado", "Vigilancia", 0, 0))
	assert.False(t, PuedeCerrarHallazgo("registrado", "Mantenimiento", 0, 0))
	assert.False(t, PuedeCerrarHallazgo("registrado", "Administración", 0, 0))
	assert.True(t, PuedeCerrarHallazgo("en_atencion", "Mantenimiento", 2, 2))
	assert.False(t, PuedeCerrarHallazgo("en_atencion", "Mantenimiento", 2, 1))
	assert.False(t, PuedeCerrarHallazgo("cerrado", "SST", 0, 0))
}

func TestTallaValida(t *testing.T) {
	letras, calzado := "letras", "calzado"
	assert.True(t, TallaValida(&letras, "M"))
	assert.False(t, TallaValida(&letras, "40"))
	assert.True(t, TallaValida(&calzado, "40"))
	assert.False(t, TallaValida(&calzado, ""))
	assert.True(t, TallaValida(nil, ""))
	assert.False(t, TallaValida(nil, "M"))
}
