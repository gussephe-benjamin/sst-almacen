package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Svc) Recursos(ctx context.Context, categoria string) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT r.id, r.nombre, r.stock, r.stock_minimo, c.id AS categoria_id, c.nombre AS categoria,
		r.stock <= r.stock_minimo AS bajo_minimo,
		(SELECT count(*) FROM solicitudes_epp e WHERE e.recurso_id = r.id AND e.estado = 'en_espera') AS en_espera
		FROM recursos r JOIN categorias_recurso c ON c.id = r.categoria_id
		WHERE r.activo AND ($1 = '' OR c.nombre = $1) ORDER BY c.id, r.nombre`, categoria)
}

type RecursoIn struct {
	CategoriaID int     `json:"categoria_id"`
	Nombre      string  `json:"nombre"`
	Stock       int     `json:"stock"`
	StockMinimo int     `json:"stock_minimo"`
	Tallas      *string `json:"tallas"` // 'letras', 'calzado' o null (talla única)
}

func (s *Svc) RecursoCrear(ctx context.Context, a *Actor, in RecursoIn) (int, error) {
	if strings.TrimSpace(in.Nombre) == "" || in.CategoriaID == 0 || in.Stock < 0 || in.StockMinimo < 0 {
		return 0, Bad("datos_incompletos", "Completa nombre, categoría, stock y mínimo.")
	}
	if in.Tallas != nil && *in.Tallas == "" {
		in.Tallas = nil
	}
	if in.Tallas != nil && Tallas[*in.Tallas] == nil {
		return 0, Bad("tallas_invalidas", "Tipo de talla no válido.")
	}
	var id int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO recursos (categoria_id, nombre, stock, stock_minimo, tallas) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			in.CategoriaID, in.Nombre, in.Stock, in.StockMinimo, in.Tallas).Scan(&id); err != nil {
			return err
		}
		return Hist(ctx, tx, a, "recurso", id, "Registró el recurso")
	})
	return id, err
}

func (s *Svc) RecursoMinimo(ctx context.Context, a *Actor, id, minimo int) error {
	if minimo < 0 {
		return Bad("minimo_invalido", "El mínimo no puede ser negativo.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE recursos SET stock_minimo=$2 WHERE id=$1`, id, minimo)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return Hist(ctx, tx, a, "recurso", id, fmt.Sprintf("Cambió el stock mínimo a %d", minimo))
	})
}

type MovIn struct {
	Tipo     string `json:"tipo"`
	Cantidad int    `json:"cantidad"`
	Nota     string `json:"nota"`
}

func (s *Svc) Movimiento(ctx context.Context, a *Actor, id int, in MovIn) error {
	if (in.Tipo != "ingreso" && in.Tipo != "salida") || in.Cantidad <= 0 {
		return Bad("movimiento_invalido", "Indica el tipo de movimiento y una cantidad mayor a 0.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var nombre string
		var stock, minimo int
		if err := tx.QueryRow(ctx, `SELECT nombre, stock, stock_minimo FROM recursos WHERE id=$1 FOR UPDATE`, id).Scan(&nombre, &stock, &minimo); err != nil {
			return notFound(err)
		}
		delta := in.Cantidad
		if in.Tipo == "salida" {
			if in.Cantidad > stock {
				return Conflict("stock_insuficiente", "No hay stock suficiente para esa salida.")
			}
			delta = -in.Cantidad
		}
		if _, err := tx.Exec(ctx, `UPDATE recursos SET stock = stock + $2 WHERE id=$1`, id, delta); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO movimientos_inventario (recurso_id, tipo, cantidad, nota, usuario_id, puesto_id) VALUES ($1,$2,$3,$4,$5,$6)`,
			id, in.Tipo, in.Cantidad, in.Nota, a.UserID, a.PuestoID); err != nil {
			return err
		}
		if err := Hist(ctx, tx, a, "recurso", id, fmt.Sprintf("Registró %s de %d", in.Tipo, in.Cantidad)); err != nil {
			return err
		}
		nuevo := stock + delta
		if in.Tipo == "ingreso" {
			var espera int
			tx.QueryRow(ctx, `SELECT count(*) FROM solicitudes_epp WHERE recurso_id=$1 AND estado='en_espera'`, id).Scan(&espera)
			if espera > 0 {
				return Notify(ctx, tx, "sst", 0, fmt.Sprintf("Ingresó stock de %s y hay %d solicitudes en espera.", nombre, espera), "inventario", id)
			}
		} else if nuevo <= minimo {
			return Notify(ctx, tx, "sst", 0, fmt.Sprintf("Stock bajo: %s (%d en stock, mínimo %d).", nombre, nuevo, minimo), "inventario", id)
		}
		return nil
	})
}
