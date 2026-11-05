package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// bajoMinimoSQL indica si un recurso (alias r) necesita reponerse: sin variantes se compara el
// total con su mínimo; con variantes, basta una talla o tipo en su mínimo o por debajo (mínimo > 0).
const bajoMinimoSQL = `CASE WHEN r.tallas IS NULL THEN r.stock <= r.stock_minimo
	ELSE EXISTS (SELECT 1 FROM recurso_variantes v WHERE v.recurso_id = r.id AND v.stock_minimo > 0 AND v.stock <= v.stock_minimo) END`

func (s *Svc) Recursos(ctx context.Context, categoria string) (json.RawMessage, error) {
	return List(ctx, s.DB, `SELECT r.id, r.nombre, r.stock, r.stock_minimo, c.id AS categoria_id, c.nombre AS categoria,
		`+bajoMinimoSQL+` AS bajo_minimo, r.tallas,
		(SELECT coalesce(json_object_agg(v.variante, v.stock), '{}') FROM recurso_variantes v WHERE v.recurso_id = r.id) AS variantes,
		(SELECT coalesce(json_object_agg(v.variante, v.stock_minimo), '{}') FROM recurso_variantes v WHERE v.recurso_id = r.id) AS minimos,
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
	Minimos     map[string]int `json:"minimos"`   // stock mínimo por variante (solo con Tallas)
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
	var variantes []string
	if in.Tallas != nil {
		// Con variantes, el stock total es la suma del stock de cada una y el mínimo va por variante.
		in.Stock, in.StockMinimo = 0, 0
		for v, n := range in.Variantes {
			if n < 0 || in.Minimos[v] < 0 || !TallaValida(in.Tallas, v) {
				return 0, Bad("variante_invalida", "Revisa el stock y el mínimo de cada talla o tipo.")
			}
			in.Stock += n
		}
		for _, v := range Tallas[*in.Tallas] {
			// Las tallas de EPP existen todas; los extintores solo tienen los tipos elegidos.
			if _, ok := in.Variantes[v]; ok || *in.Tallas != "extintor" {
				variantes = append(variantes, v)
			}
		}
		if len(variantes) == 0 {
			return 0, Bad("variante_requerida", "Elige al menos un tipo de extintor.")
		}
	}
	var id int
	err := s.Tx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO recursos (categoria_id, nombre, stock, stock_minimo, tallas) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
			in.CategoriaID, in.Nombre, in.Stock, in.StockMinimo, in.Tallas).Scan(&id); err != nil {
			return err
		}
		for _, v := range variantes {
			if _, err := tx.Exec(ctx, `INSERT INTO recurso_variantes (recurso_id, variante, stock, stock_minimo) VALUES ($1,$2,$3,$4)`,
				id, v, in.Variantes[v], in.Minimos[v]); err != nil {
				return err
			}
		}
		return Hist(ctx, tx, a, "recurso", id, "Registró el recurso")
	})
	return id, err
}

type MinimoIn struct {
	StockMinimo int            `json:"stock_minimo"`
	Minimos     map[string]int `json:"minimos"` // por variante, en recursos con tallas o tipos
}

func (s *Svc) RecursoMinimo(ctx context.Context, a *Actor, id int, in MinimoIn) error {
	if in.StockMinimo < 0 {
		return Bad("minimo_invalido", "El mínimo no puede ser negativo.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var tipo *string
		if err := tx.QueryRow(ctx, `SELECT tallas FROM recursos WHERE id=$1 FOR UPDATE`, id).Scan(&tipo); err != nil {
			return notFound(err)
		}
		if tipo == nil {
			if _, err := tx.Exec(ctx, `UPDATE recursos SET stock_minimo=$2 WHERE id=$1`, id, in.StockMinimo); err != nil {
				return err
			}
			return Hist(ctx, tx, a, "recurso", id, fmt.Sprintf("Cambió el stock mínimo a %d", in.StockMinimo))
		}
		var cambios []string
		for _, v := range Tallas[*tipo] {
			n, ok := in.Minimos[v]
			if !ok {
				continue
			}
			if n < 0 {
				return Bad("minimo_invalido", "El mínimo no puede ser negativo.")
			}
			tag, err := tx.Exec(ctx, `UPDATE recurso_variantes SET stock_minimo=$3 WHERE recurso_id=$1 AND variante=$2 AND stock_minimo <> $3`, id, v, n)
			if err != nil {
				return err
			}
			if tag.RowsAffected() > 0 {
				cambios = append(cambios, fmt.Sprintf("%s: %d", v, n))
			}
		}
		if len(cambios) == 0 {
			return nil
		}
		return Hist(ctx, tx, a, "recurso", id, "Cambió el stock mínimo ("+strings.Join(cambios, ", ")+")")
	})
}

// MovItem es la cantidad de una talla o tipo dentro de un movimiento.
type MovItem struct {
	Variante string `json:"variante"`
	Cantidad int    `json:"cantidad"`
}

type MovIn struct {
	Tipo     string    `json:"tipo"`
	Cantidad int       `json:"cantidad"` // recursos sin variantes
	Nota     string    `json:"nota"`
	Items    []MovItem `json:"items"` // recursos con variantes: una o varias tallas o tipos a la vez
}

// Movimiento registra un ingreso o una salida. En recursos con variantes se pueden mover varias
// tallas o tipos en una sola operación: se aplican todas o ninguna.
func (s *Svc) Movimiento(ctx context.Context, a *Actor, id int, in MovIn) error {
	if in.Tipo != "ingreso" && in.Tipo != "salida" {
		return Bad("movimiento_invalido", "Indica el tipo de movimiento.")
	}
	return s.Tx(ctx, func(tx pgx.Tx) error {
		var nombre string
		var tipo *string
		var stock, minimo int
		if err := tx.QueryRow(ctx, `SELECT nombre, tallas, stock, stock_minimo FROM recursos WHERE id=$1 FOR UPDATE`, id).Scan(&nombre, &tipo, &stock, &minimo); err != nil {
			return notFound(err)
		}
		items := in.Items
		if tipo == nil {
			items = []MovItem{{Cantidad: in.Cantidad}}
		} else if len(items) == 0 {
			return Bad("variante_requerida", "Elige al menos una talla o tipo.")
		}
		var avisos, detalle []string
		total := 0
		vistos := map[string]bool{}
		for _, it := range items {
			if it.Cantidad <= 0 {
				return Bad("movimiento_invalido", "Indica una cantidad mayor a 0.")
			}
			if tipo != nil && (!TallaValida(tipo, it.Variante) || vistos[it.Variante]) {
				return Bad("variante_requerida", "Revisa las tallas o tipos elegidos.")
			}
			vistos[it.Variante] = true
			delta := it.Cantidad
			if in.Tipo == "salida" {
				delta = -it.Cantidad
			}
			total += delta
			if tipo == nil {
				if stock+delta < 0 {
					return Conflict("stock_insuficiente", "No hay stock suficiente para esa salida.")
				}
				if in.Tipo == "salida" && stock+delta <= minimo {
					avisos = append(avisos, fmt.Sprintf("Stock bajo: %s (%d en stock, mínimo %d).", nombre, stock+delta, minimo))
				}
				detalle = append(detalle, fmt.Sprint(it.Cantidad))
				continue
			}
			var vStock, vMin int
			err := tx.QueryRow(ctx, `INSERT INTO recurso_variantes (recurso_id, variante) VALUES ($1,$2)
				ON CONFLICT (recurso_id, variante) DO UPDATE SET stock = recurso_variantes.stock RETURNING stock, stock_minimo`, id, it.Variante).Scan(&vStock, &vMin)
			if err != nil {
				return err
			}
			if vStock+delta < 0 {
				return Conflict("stock_insuficiente", fmt.Sprintf("No hay stock suficiente de %s: hay %d.", it.Variante, vStock))
			}
			if _, err := tx.Exec(ctx, `UPDATE recurso_variantes SET stock = stock + $3 WHERE recurso_id=$1 AND variante=$2`, id, it.Variante, delta); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO movimientos_inventario (recurso_id, tipo, cantidad, nota, usuario_id, puesto_id, variante) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				id, in.Tipo, it.Cantidad, in.Nota, a.UserID, a.PuestoID, it.Variante); err != nil {
				return err
			}
			if in.Tipo == "salida" && vMin > 0 && vStock+delta <= vMin {
				avisos = append(avisos, fmt.Sprintf("Stock bajo: %s %s (%d en stock, mínimo %d).", nombre, it.Variante, vStock+delta, vMin))
			}
			detalle = append(detalle, fmt.Sprintf("%d de %s", it.Cantidad, it.Variante))
		}
		if _, err := tx.Exec(ctx, `UPDATE recursos SET stock = stock + $2 WHERE id=$1`, id, total); err != nil {
			return err
		}
		if tipo == nil {
			if _, err := tx.Exec(ctx, `INSERT INTO movimientos_inventario (recurso_id, tipo, cantidad, nota, usuario_id, puesto_id) VALUES ($1,$2,$3,$4,$5,$6)`,
				id, in.Tipo, in.Cantidad, in.Nota, a.UserID, a.PuestoID); err != nil {
				return err
			}
		}
		if err := Hist(ctx, tx, a, "recurso", id, fmt.Sprintf("Registró %s de %s", in.Tipo, strings.Join(detalle, ", "))); err != nil {
			return err
		}
		if in.Tipo == "ingreso" {
			var espera int
			tx.QueryRow(ctx, `SELECT count(*) FROM solicitudes_epp WHERE recurso_id=$1 AND estado='en_espera'`, id).Scan(&espera)
			if espera > 0 {
				return Notify(ctx, tx, "sst", 0, fmt.Sprintf("Ingresó stock de %s y hay %d solicitudes en espera.", nombre, espera), "inventario", id)
			}
		}
		for _, m := range avisos {
			if err := Notify(ctx, tx, "sst", 0, m, "inventario", id); err != nil {
				return err
			}
		}
		return nil
	})
}
