import { useState } from 'react'
import type { Row } from '../api/client'
import { Btn, plural, Banner, Field, Loading, Modal, Page, Tabs, useAction, useApi } from '../components/ui'
import { TALLAS } from './epp'

// Etiqueta de la variante según el tipo: talla en EPP, tipo en extintores.
const nombreVariante = (tipo?: string | null) => (tipo === 'extintor' ? 'Tipo' : 'Talla')
const etiqueta = (v: string) => (v === '' ? 'Sin talla' : v)

/** Variantes de un recurso en orden (XS…XXL, 35…45, PQS…), con la variante '' (stock anterior) al final si existe. */
function variantesDe(r: Row): [string, number][] {
  const stock: Record<string, number> = r.variantes ?? {}
  const orden = [...(TALLAS[r.tallas] ?? []), ...('' in stock ? [''] : [])]
  return orden.map((v) => [v, stock[v] ?? 0])
}

export function Inventario() {
  const cat = useApi<Row>('/catalogos')
  const [tab, setTab] = useState('')
  const q = useApi(`/recursos?categoria=${encodeURIComponent(tab)}`)
  const act = useAction()
  const [mov, setMov] = useState<{ r: Row; tipo: string } | null>(null)
  const [min, setMin] = useState<Row | null>(null)
  const [nuevo, setNuevo] = useState(false)
  const [v, setV] = useState({ cantidad: 1, nota: '', stock_minimo: 0, variante: null as string | null })
  const vacio = { nombre: '', categoria_id: 1, stock: 0, stock_minimo: 0, tallas: '', variantes: {} as Record<string, number> }
  const [n, setN] = useState(vacio)
  const categoria = (cat.data?.categorias ?? []).find((c: Row) => c.id === n.categoria_id)?.nombre
  const esEpp = categoria === 'EPP'
  const esExtintor = categoria === 'Extintores'
  const tallasN = esEpp || esExtintor ? n.tallas : ''
  const low = (q.data ?? []).filter((r: Row) => r.bajo_minimo)

  const abrirMov = (r: Row, tipo: string) => {
    setV({ ...v, cantidad: 1, nota: '', variante: null })
    setMov({ r, tipo })
  }
  const stockMov = mov && (mov.r.tallas ? (v.variante === null ? null : mov.r.variantes?.[v.variante] ?? 0) : mov.r.stock)

  return (
    <Page title="Inventario" actions={<Btn className="btn btn-primary" onClick={() => { setN(vacio); setNuevo(true) }}>Nuevo recurso</Btn>}>
      <Tabs value={tab} onChange={setTab} tabs={[['', 'Todos'], ...((cat.data?.categorias ?? []) as Row[]).map((c): [string, string] => [c.nombre, c.nombre])]} />
      {low.length > 0 && <Banner kind="warn"><b>{plural(low.length, 'recurso', 'recursos')} en o bajo el mínimo:</b> {low.map((r: Row) => r.nombre).join(', ')}.</Banner>}
      <Loading q={q} />
      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Recurso</th><th>Categoría</th><th>Stock</th><th>Mínimo</th><th></th></tr></thead>
          <tbody>
            {(q.data ?? []).map((r: Row) => (
              <tr key={r.id}>
                <td>
                  {r.nombre}
                  {r.tallas && (
                    <div className="mt-1 flex flex-wrap gap-1">
                      {variantesDe(r).map(([t, s]) => (
                        <span key={t} className={`rounded px-1.5 py-0.5 text-[11px] ${s > 0 ? 'bg-emerald-50 text-emerald-800' : 'bg-slate-100 text-slate-400'}`}>{etiqueta(t)}: {s}</span>
                      ))}
                    </div>
                  )}
                  {r.en_espera > 0 && <div className="text-xs text-amber-700">{r.en_espera} solicitudes en espera</div>}
                </td>
                <td>{r.categoria}</td>
                <td className={r.bajo_minimo ? 'font-bold text-red-700' : ''}>{r.stock}</td>
                <td>{r.stock_minimo}</td>
                <td className="whitespace-nowrap text-right">
                  <Btn className="btn btn-sm" onClick={() => abrirMov(r, 'ingreso')}>Ingreso</Btn>{' '}
                  <Btn className="btn btn-sm" onClick={() => abrirMov(r, 'salida')}>Salida</Btn>{' '}
                  <Btn className="btn btn-sm" onClick={() => { setV({ ...v, stock_minimo: r.stock_minimo }); setMin(r) }}>Mínimo</Btn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {q.data?.length === 0 && <p className="muted p-4">{tab ? `No hay recursos en ${tab}.` : 'Aún no hay recursos. Registra el primero con “Nuevo recurso”.'}</p>}
      </div>
      {mov && (
        <Modal title={`${mov.tipo === 'ingreso' ? 'Ingreso' : 'Salida'}: ${mov.r.nombre}`} onClose={() => setMov(null)}
          footer={<Btn className="btn btn-primary" onClick={async () => {
            if (await act(`/recursos/${mov.r.id}/movimientos`, { tipo: mov.tipo, cantidad: v.cantidad, nota: v.nota, variante: v.variante ?? '' }, 'Movimiento registrado.')) setMov(null)
          }}>Registrar</Btn>}>
          {mov.r.tallas && (
            <Field label={nombreVariante(mov.r.tallas)}>
              <div className="flex flex-wrap gap-1.5">
                {variantesDe(mov.r).map(([t, s]) => (
                  <button key={t} type="button" onClick={() => setV({ ...v, variante: t })}
                    className={`rounded-md border px-2 py-1.5 text-sm ${v.variante === t ? 'border-blue-700 bg-blue-700 text-white' : 'border-slate-300 bg-white'}`}>
                    {etiqueta(t)} <span className="opacity-70">({s})</span>
                  </button>
                ))}
              </div>
            </Field>
          )}
          <p className="muted">{stockMov === null ? `Elige la ${nombreVariante(mov.r.tallas).toLowerCase()}.` : `Stock actual${v.variante !== null ? ` de ${etiqueta(v.variante)}` : ''}: ${stockMov}`}</p>
          <Field label="Cantidad"><input type="number" min={1} className="inp" value={v.cantidad} onChange={(e) => setV({ ...v, cantidad: +e.target.value })} /></Field>
          <Field label="Nota"><input className="inp" value={v.nota} onChange={(e) => setV({ ...v, nota: e.target.value })} /></Field>
        </Modal>
      )}
      {min && (
        <Modal title={`Stock mínimo: ${min.nombre}`} onClose={() => setMin(null)}
          footer={<Btn className="btn btn-primary" onClick={async () => { if (await act(`/recursos/${min.id}`, { stock_minimo: v.stock_minimo }, 'Mínimo actualizado.', 'PATCH')) setMin(null) }}>Guardar</Btn>}>
          <Field label="Mínimo (stock total)"><input type="number" min={0} className="inp" value={v.stock_minimo} onChange={(e) => setV({ ...v, stock_minimo: +e.target.value })} /></Field>
        </Modal>
      )}
      {nuevo && (
        <Modal title="Nuevo recurso" onClose={() => setNuevo(false)}
          footer={<Btn className="btn btn-primary" onClick={async () => {
            if (await act('/recursos', { ...n, tallas: tallasN, variantes: tallasN ? n.variantes : {} }, 'Recurso registrado.')) setNuevo(false)
          }}>Guardar</Btn>}>
          <Field label="Nombre"><input className="inp" value={n.nombre} onChange={(e) => setN({ ...n, nombre: e.target.value })} /></Field>
          <Field label="Categoría">
            <select className="inp" value={n.categoria_id} onChange={(e) => {
              const nombre = (cat.data?.categorias ?? []).find((c: Row) => c.id === +e.target.value)?.nombre
              setN({ ...n, categoria_id: +e.target.value, tallas: nombre === 'Extintores' ? 'extintor' : '', variantes: {} })
            }}>
              {(cat.data?.categorias ?? []).map((c: Row) => <option key={c.id} value={c.id}>{c.nombre}</option>)}
            </select>
          </Field>
          {esEpp && (
            <Field label="Tallas que puede elegir el trabajador">
              <select className="inp" value={n.tallas} onChange={(e) => setN({ ...n, tallas: e.target.value, variantes: {} })}>
                <option value="">Talla única</option>
                <option value="letras">Por letras (XS a XXL)</option>
                <option value="calzado">Calzado (35 a 45)</option>
              </select>
            </Field>
          )}
          {esExtintor && (
            <Field label="Tipo de extintor">
              <select className="inp" value={n.tallas} onChange={(e) => setN({ ...n, tallas: e.target.value, variantes: {} })}>
                <option value="extintor">Por tipo (PQS, CO2, Agua presurizada)</option>
                <option value="">Sin tipo</option>
              </select>
            </Field>
          )}
          {tallasN ? (
            <Field label={`Stock inicial por ${nombreVariante(tallasN).toLowerCase()}`}>
              <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
                {(TALLAS[tallasN] ?? []).map((t) => (
                  <label key={t} className="flex items-center gap-1.5 rounded-md border border-slate-200 px-2 py-1 text-sm">
                    <span className="min-w-8 font-medium">{t}</span>
                    <input type="number" min={0} className="inp h-8 w-full min-w-0 px-1.5" value={n.variantes[t] ?? 0}
                      onChange={(e) => setN({ ...n, variantes: { ...n.variantes, [t]: Math.max(0, +e.target.value) } })} />
                  </label>
                ))}
              </div>
              <p className="muted mt-1">Total: {Object.values(n.variantes).reduce((a, b) => a + b, 0)}. Deja en 0 las que no tengas.</p>
            </Field>
          ) : null}
          <div className="grid grid-cols-2 gap-3">
            {!tallasN && <Field label="Stock inicial"><input type="number" min={0} className="inp" value={n.stock} onChange={(e) => setN({ ...n, stock: +e.target.value })} /></Field>}
            <Field label="Mínimo (stock total)"><input type="number" min={0} className="inp" value={n.stock_minimo} onChange={(e) => setN({ ...n, stock_minimo: +e.target.value })} /></Field>
          </div>
        </Modal>
      )}
    </Page>
  )
}
