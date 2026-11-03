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
		r.stock <= r.stock_minimo AS bajo_minimo, r.tallas,
		(SELECT coalesce(json_object_agg(v.variante, v.stock), '{}') FROM recurso_variantes v WHERE v.recurso_id = r.id) AS variantes,
		(SELECT count(*) FROM solicitudes_epp e WHERE e.recurso_id = r.id AND e.estado = 'en_espera') AS en_espera
		FROM recursos r JOIN categorias_recurso c ON c.id = r.categoria_id
		WHERE r.activo AND ($1 = '' OR c.nombre = $1) ORDER BY c.id, r.nombre`, categoria)
}

type RecursoIn struct {
	CategoriaID int            `json:"categoria_id"`
	Nombre      string         `json:"nombre"`
	Stock       int            `json:"stock"`
	StockMinimo int            `json:"stock_minimo"`
	Tallas      *string        `json:"tallas"`    // 'letras', 'calzado', 'extintor' o null (sin variantes)
	Variantes   map[string]int `json:"variantes"` // stock inicial por variante (solo con Tallas)
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
	if in.Tallas != nil {
		// Con variantes, el stock total es la suma del stock de cada una.
		in.Stock = 0
		for v, n := range in.Variantes {
			if n < 0 || !TallaValida(in.Tallas, v) {
				return 0, Bad("variante_invalida", "Revisa el stock de cada talla o tipo.")
			}
			in.Stock += n
		}
	}
	var id int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO recursos (categoria_id, nombre, stock, stock_minimo, tallas) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			in.CategoriaID, in.Nombre, in.Stock, in.StockMinimo, in.Tallas).Scan(&id); err != nil {
			return err
		}
		if in.Tallas != nil {
			for _, v := range Tallas[*in.Tallas] {
				if _, err := tx.Exec(ctx, `INSERT INTO recurso_variantes (recurso_id, variante, stock) VALUES ($1,$2,$3)`, id, v, in.Variantes[v]); err != nil {
					return err
				}
			}
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
	Variante string `json:"variante"` // talla o tipo; obligatorio si el recurso tiene variantes
}

func (s *Svc) Movimiento(ctx context.Context, a *Actor, id int, in MovIn) error {
	if (in.Tipo != "ingreso" && in.Tipo != "salida") || in.Cantidad <= 0 {
		return Bad("movimiento_invalido", "Indica el tipo de movimiento y una cantidad mayor a 0.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var nombre string
		var tipo *string
		var stock, minimo int
		if err := tx.QueryRow(ctx, `SELECT nombre, tallas, stock, stock_minimo FROM recursos WHERE id=$1 FOR UPDATE`, id).Scan(&nombre, &tipo, &stock, &minimo); err != nil {
			return notFound(err)
		}
		var variante *string
		disponible := stock
		if tipo != nil {
			// La variante debe ser del tipo del recurso, o '' (stock "Sin talla" anterior a las variantes).
			if err := tx.QueryRow(ctx, `SELECT stock FROM recurso_variantes WHERE recurso_id=$1 AND variante=$2 FOR UPDATE`, id, in.Variante).Scan(&disponible); err != nil {
				if in.Variante == "" || !TallaValida(tipo, in.Variante) {
					return Bad("variante_requerida", "Elige la talla o el tipo.")
				}
				if _, err := tx.Exec(ctx, `INSERT INTO recurso_variantes (recurso_id, variante) VALUES ($1,$2)`, id, in.Variante); err != nil {
					return err
				}
				disponible = 0
			}
			variante = &in.Variante
		}
		delta := in.Cantidad
		if in.Tipo == "salida" {
			if in.Cantidad > disponible {
				return Conflict("stock_insuficiente", "No hay stock suficiente para esa salida.")
			}
			delta = -in.Cantidad
		}
		if _, err := tx.Exec(ctx, `UPDATE recursos SET stock = stock + $2 WHERE id=$1`, id, delta); err != nil {
			return err
		}
		if variante != nil {
			if _, err := tx.Exec(ctx, `UPDATE recurso_variantes SET stock = stock + $3 WHERE recurso_id=$1 AND variante=$2`, id, *variante, delta); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO movimientos_inventario (recurso_id, tipo, cantidad, nota, usuario_id, puesto_id, variante) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			id, in.Tipo, in.Cantidad, in.Nota, a.UserID, a.PuestoID, variante); err != nil {
			return err
		}
		detalle := ""
		if variante != nil && *variante != "" {
			detalle = " (" + *variante + ")"
		}
		if err := Hist(ctx, tx, a, "recurso", id, fmt.Sprintf("Registró %s de %d%s", in.Tipo, in.Cantidad, detalle)); err != nil {
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
