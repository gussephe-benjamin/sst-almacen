import { useState } from 'react'
import { X } from 'lucide-react'
import type { Row } from '../api/client'
import { Btn, plural, Banner, Field, Loading, Modal, Page, Tabs, toast, useAction, useApi } from '../components/ui'
import { TALLAS } from './epp'

// Etiqueta de la variante según el tipo: talla en EPP, tipo en extintores.
const nombreVariante = (tipo?: string | null) => (tipo === 'extintor' ? 'Tipo' : 'Talla')

/** Variantes que tiene un recurso, en orden (XS…XXL, 35…45, PQS…), con su stock y su mínimo. */
function variantesDe(r: Row): { v: string; stock: number; min: number; bajo: boolean }[] {
  const stock: Record<string, number> = r.variantes ?? {}
  const minimos: Record<string, number> = r.minimos ?? {}
  return (TALLAS[r.tallas] ?? []).filter((v) => v in stock).map((v) => {
    const min = minimos[v] ?? 0
    return { v, stock: stock[v], min, bajo: min > 0 && stock[v] <= min }
  })
}

/** Lo que está en o bajo el mínimo: el recurso o, si tiene variantes, cuáles. */
const faltante = (r: Row) => {
  const bajas = variantesDe(r).filter((x) => x.bajo).map((x) => x.v)
  return bajas.length ? `${r.nombre} (${bajas.join(', ')})` : r.nombre
}

type Fila = { stock: number; min: number }

/** Stock inicial y mínimo de cada talla o tipo. En extintores se marcan los tipos que se tienen. */
function GrillaVariantes({ tipo, filas, onChange }: { tipo: string; filas: Record<string, Fila>; onChange: (f: Record<string, Fila>) => void }) {
  const elegible = tipo === 'extintor'
  const total = Object.values(filas).reduce((a, f) => a + f.stock, 0)
  const set = (v: string, k: keyof Fila, n: number) => onChange({ ...filas, [v]: { ...filas[v], [k]: Math.max(0, n || 0) } })
  return (
    <Field label={elegible ? 'Tipos de extintor' : 'Stock inicial y mínimo por talla'}>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
        {TALLAS[tipo].map((v) => {
          const f = filas[v]
          return (
            <div key={v} className={`rounded-md border px-2 py-1.5 ${f ? 'border-slate-300' : 'border-dashed border-slate-200'}`}>
              {elegible ? (
                <label className="flex items-center gap-1.5 text-sm font-medium">
                  <input type="checkbox" checked={!!f} onChange={(e) => {
                    const next = { ...filas }
                    if (e.target.checked) next[v] = { stock: 0, min: 0 }
                    else delete next[v]
                    onChange(next)
                  }} /> {v}
                </label>
              ) : <div className="text-sm font-medium">{v}</div>}
              {f && (
                <div className="mt-1 grid grid-cols-2 gap-1.5">
                  <label className="text-[11px] text-slate-500">Stock
                    <input type="number" min={0} className="inp h-8 w-full min-w-0 px-1.5" value={f.stock} onChange={(e) => set(v, 'stock', +e.target.value)} />
                  </label>
                  <label className="text-[11px] text-slate-500">Mínimo
                    <input type="number" min={0} className="inp h-8 w-full min-w-0 px-1.5" value={f.min} onChange={(e) => set(v, 'min', +e.target.value)} />
                  </label>
                </div>
              )}
            </div>
          )
        })}
      </div>
      <p className="muted mt-1">Total: {total}. {elegible ? 'Marca los tipos que tienes.' : 'Deja en 0 las tallas que no tengas.'} Con mínimo 0 no se avisa.</p>
    </Field>
  )
}

/** Todas las tallas en 0; en extintores, ningún tipo marcado. */
const filasIniciales = (tipo: string): Record<string, Fila> =>
  tipo && tipo !== 'extintor' ? Object.fromEntries(TALLAS[tipo].map((v) => [v, { stock: 0, min: 0 }])) : {}

export function Inventario() {
  const cat = useApi<Row>('/catalogos')
  const [tab, setTab] = useState('')
  const q = useApi(`/recursos?categoria=${encodeURIComponent(tab)}`)
  const act = useAction()
  const [mov, setMov] = useState<{ r: Row; tipo: string } | null>(null)
  const [min, setMin] = useState<Row | null>(null)
  const [nuevo, setNuevo] = useState(false)
  // Movimiento: cantidad (sin variantes) o cantidad por cada talla o tipo elegido.
  const [v, setV] = useState({ cantidad: 1, nota: '', items: {} as Record<string, number> })
  const [minimos, setMinimos] = useState({ total: 0, porVariante: {} as Record<string, number> })
  const vacio = { nombre: '', categoria_id: 1, stock: 0, stock_minimo: 0, tallas: '', filas: {} as Record<string, Fila> }
  const [n, setN] = useState(vacio)
  const categoria = (cat.data?.categorias ?? []).find((c: Row) => c.id === n.categoria_id)?.nombre
  const esEpp = categoria === 'EPP'
  const tallasN = esEpp || categoria === 'Extintores' ? n.tallas : ''
  const low = (q.data ?? []).filter((r: Row) => r.bajo_minimo)

  const abrirMov = (r: Row, tipo: string) => {
    setV({ cantidad: 1, nota: '', items: {} })
    setMov({ r, tipo })
  }
  const abrirMin = (r: Row) => {
    setMinimos({ total: r.stock_minimo, porVariante: { ...(r.minimos ?? {}) } })
    setMin(r)
  }
  const registrarMov = async () => {
    if (!mov) return
    const items = Object.entries(v.items).map(([variante, cantidad]) => ({ variante, cantidad }))
    if (mov.r.tallas && !items.length) return toast(`Elige al menos una ${nombreVariante(mov.r.tallas).toLowerCase()}.`, true)
    const ok = await act(`/recursos/${mov.r.id}/movimientos`, { tipo: mov.tipo, cantidad: v.cantidad, nota: v.nota, items }, 'Movimiento registrado.')
    if (ok) setMov(null)
  }
  const guardarNuevo = async () => {
    const variantes = Object.fromEntries(Object.entries(n.filas).map(([k, f]) => [k, f.stock]))
    const mins = Object.fromEntries(Object.entries(n.filas).map(([k, f]) => [k, f.min]))
    if (tallasN === 'extintor' && !Object.keys(n.filas).length) return toast('Marca al menos un tipo de extintor.', true)
    const body = { nombre: n.nombre, categoria_id: n.categoria_id, stock: n.stock, stock_minimo: n.stock_minimo, tallas: tallasN, variantes: tallasN ? variantes : {}, minimos: tallasN ? mins : {} }
    if (await act('/recursos', body, 'Recurso registrado.')) setNuevo(false)
  }

  return (
    <Page title="Inventario" actions={<Btn className="btn btn-primary" onClick={() => { setN(vacio); setNuevo(true) }}>Nuevo recurso</Btn>}>
      <Tabs value={tab} onChange={setTab} tabs={[['', 'Todos'], ...((cat.data?.categorias ?? []) as Row[]).map((c): [string, string] => [c.nombre, c.nombre])]} />
      {low.length > 0 && <Banner kind="warn"><b>{plural(low.length, 'recurso', 'recursos')} en o bajo el mínimo:</b> {low.map(faltante).join(', ')}.</Banner>}
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
                      {variantesDe(r).map((x) => (
                        <span key={x.v} title={x.min ? `Mínimo ${x.min}` : 'Sin mínimo'}
                          className={`rounded px-1.5 py-0.5 text-[11px] ${x.bajo ? 'bg-red-50 font-semibold text-red-700' : x.stock > 0 ? 'bg-emerald-50 text-emerald-800' : 'bg-slate-100 text-slate-400'}`}>{x.v}: {x.stock}</span>
                      ))}
                    </div>
                  )}
                  {r.en_espera > 0 && <div className="text-xs text-amber-700">{r.en_espera} solicitudes en espera</div>}
                </td>
                <td>{r.categoria}</td>
                <td className={r.bajo_minimo ? 'font-bold text-red-700' : ''}>{r.stock}</td>
                <td>{r.tallas ? <span className="text-xs text-slate-500">Por {nombreVariante(r.tallas).toLowerCase()}</span> : r.stock_minimo}</td>
                <td className="whitespace-nowrap text-right">
                  <Btn className="btn btn-sm" onClick={() => abrirMov(r, 'ingreso')}>Ingreso</Btn>{' '}
                  <Btn className="btn btn-sm" onClick={() => abrirMov(r, 'salida')}>Salida</Btn>{' '}
                  <Btn className="btn btn-sm" onClick={() => abrirMin(r)}>Mínimo</Btn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {q.data?.length === 0 && <p className="muted p-4">{tab ? `No hay recursos en ${tab}.` : 'Aún no hay recursos. Registra el primero con “Nuevo recurso”.'}</p>}
      </div>
      {mov && (
        <Modal title={`${mov.tipo === 'ingreso' ? 'Ingreso' : 'Salida'}: ${mov.r.nombre}`} onClose={() => setMov(null)}
          footer={<Btn className="btn btn-primary" onClick={registrarMov}>Registrar</Btn>}>
          {mov.r.tallas ? <>
            <Field label={`${nombreVariante(mov.r.tallas)}s (puedes elegir varias)`}>
              <div className="flex flex-wrap gap-1.5">
                {variantesDe(mov.r).map((x) => {
                  const sel = x.v in v.items
                  return (
                    <button key={x.v} type="button" onClick={() => setV((p) => {
                      const items = { ...p.items }
                      if (x.v in items) delete items[x.v]
                      else items[x.v] = 1
                      return { ...p, items }
                    })}
                      className={`rounded-md border px-2 py-1.5 text-sm ${sel ? 'border-blue-700 bg-blue-700 text-white' : 'border-slate-300 bg-white'}`}>
                      {x.v} <span className="opacity-70">({x.stock})</span>
                    </button>
                  )
                })}
              </div>
            </Field>
            {Object.keys(v.items).length === 0 ? <p className="muted">Toca cada {nombreVariante(mov.r.tallas).toLowerCase()} y escribe su cantidad.</p> : (
              <div className="space-y-1.5">
                {variantesDe(mov.r).filter((x) => x.v in v.items).map((x) => (
                  <div key={x.v} className="flex items-center gap-2 rounded-md border border-slate-200 px-2 py-1.5">
                    <b className="min-w-16 text-sm">{x.v}</b>
                    <input type="number" min={1} aria-label={`Cantidad de ${x.v}`} className="inp h-8 w-24" value={v.items[x.v]}
                      onChange={(e) => { const n = Math.max(0, +e.target.value || 0); setV((p) => ({ ...p, items: { ...p.items, [x.v]: n } })) }} />
                    <span className="flex-1 text-xs text-slate-500">Stock {x.stock} → {x.stock + (mov.tipo === 'ingreso' ? 1 : -1) * (v.items[x.v] || 0)}</span>
                    <button type="button" className="text-slate-400 hover:text-red-600" title="Quitar" onClick={() => setV((p) => {
                      const items = { ...p.items }
                      delete items[x.v]
                      return { ...p, items }
                    })}><X size={14} /></button>
                  </div>
                ))}
                <p className="text-right text-xs text-slate-500">Total: {Object.values(v.items).reduce((a, b) => a + b, 0)}</p>
              </div>
            )}
          </> : <>
            <p className="muted">Stock actual: {mov.r.stock}</p>
            <Field label="Cantidad"><input type="number" min={1} className="inp" value={v.cantidad} onChange={(e) => setV({ ...v, cantidad: +e.target.value })} /></Field>
          </>}
          <Field label="Nota"><input className="inp" value={v.nota} onChange={(e) => setV({ ...v, nota: e.target.value })} /></Field>
        </Modal>
      )}
      {min && (
        <Modal title={`Stock mínimo: ${min.nombre}`} onClose={() => setMin(null)}
          footer={<Btn className="btn btn-primary" onClick={async () => {
            if (await act(`/recursos/${min.id}`, { stock_minimo: minimos.total, minimos: minimos.porVariante }, 'Mínimo actualizado.', 'PATCH')) setMin(null)
          }}>Guardar</Btn>}>
          {min.tallas ? <>
            <p className="muted">Te avisamos cuando una {nombreVariante(min.tallas).toLowerCase()} llegue a su mínimo. Con 0 no se avisa.</p>
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
              {variantesDe(min).map((x) => (
                <label key={x.v} className="rounded-md border border-slate-200 px-2 py-1 text-sm">
                  <span className="font-medium">{x.v}</span> <span className="text-xs text-slate-500">({x.stock})</span>
                  <input type="number" min={0} className="inp mt-0.5 h-8 w-full min-w-0 px-1.5" value={minimos.porVariante[x.v] ?? 0}
                    onChange={(e) => setMinimos({ ...minimos, porVariante: { ...minimos.porVariante, [x.v]: Math.max(0, +e.target.value || 0) } })} />
                </label>
              ))}
            </div>
          </> : <Field label="Mínimo"><input type="number" min={0} className="inp" value={minimos.total} onChange={(e) => setMinimos({ ...minimos, total: +e.target.value })} /></Field>}
        </Modal>
      )}
      {nuevo && (
        <Modal title="Nuevo recurso" onClose={() => setNuevo(false)} footer={<Btn className="btn btn-primary" onClick={guardarNuevo}>Guardar</Btn>}>
          <Field label="Nombre"><input className="inp" value={n.nombre} onChange={(e) => setN({ ...n, nombre: e.target.value })} /></Field>
          <Field label="Categoría">
            <select className="inp" value={n.categoria_id} onChange={(e) => {
              const nombre = (cat.data?.categorias ?? []).find((c: Row) => c.id === +e.target.value)?.nombre
              const tallas = nombre === 'Extintores' ? 'extintor' : ''
              setN({ ...n, categoria_id: +e.target.value, tallas, filas: filasIniciales(tallas) })
            }}>
              {(cat.data?.categorias ?? []).map((c: Row) => <option key={c.id} value={c.id}>{c.nombre}</option>)}
            </select>
          </Field>
          {esEpp && (
            <Field label="Tallas que puede elegir el trabajador">
              <select className="inp" value={n.tallas} onChange={(e) => setN({ ...n, tallas: e.target.value, filas: filasIniciales(e.target.value) })}>
                <option value="">Talla única</option>
                <option value="letras">Por letras (XS a XXL)</option>
                <option value="calzado">Calzado (35 a 45)</option>
              </select>
            </Field>
          )}
          {tallasN ? <GrillaVariantes tipo={tallasN} filas={n.filas} onChange={(filas) => setN({ ...n, filas })} /> : (
            <div className="grid grid-cols-2 gap-3">
              <Field label="Stock inicial"><input type="number" min={0} className="inp" value={n.stock} onChange={(e) => setN({ ...n, stock: +e.target.value })} /></Field>
              <Field label="Mínimo"><input type="number" min={0} className="inp" value={n.stock_minimo} onChange={(e) => setN({ ...n, stock_minimo: +e.target.value })} /></Field>
            </div>
          )}
        </Modal>
      )}
    </Page>
  )
}
