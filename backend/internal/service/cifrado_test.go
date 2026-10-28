package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCifradoPasswordAsignada(t *testing.T) {
	s := &Svc{PwKey: ClavePasswords("secreto-de-prueba-con-mas-de-32-caracteres")}
	c, err := s.cifrar("clave-inicial-123")
	require.NoError(t, err)
	require.NotContains(t, c, "clave-inicial-123")

	pw, err := s.descifrar(c)
	require.NoError(t, err)
	require.Equal(t, "clave-inicial-123", pw)

	// Con otro secreto no se puede leer.
	otro := &Svc{PwKey: ClavePasswords("otro-secreto-distinto-con-mas-de-32-caracteres")}
	_, err = otro.descifrar(c)
	require.Error(t, err)
}
