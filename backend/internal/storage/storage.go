package storage

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

type Store struct{ Dir string }

// Save escribe el contenido en disco y devuelve la ruta relativa.
func (s *Store) Save(r io.Reader, ext string) (string, int, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", 0, err
	}
	b := make([]byte, 16)
	rand.Read(b)
	name := hex.EncodeToString(b) + ext
	f, err := os.Create(filepath.Join(s.Dir, name))
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, r)
	return name, int(n), err
}

func (s *Store) Path(name string) string { return filepath.Join(s.Dir, filepath.Base(name)) }
