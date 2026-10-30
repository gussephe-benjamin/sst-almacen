import { useState } from 'react'
import type { Row } from '../api/client'
import { Btn, plural, Banner, Field, Loading, Modal, Page, Tabs, useAction, useApi } from '../components/ui'

export function Inventario() {
  const cat = useApi<Row>('/catalogos')
  const [tab, setTab] = useState('')
  const q = useApi(`/recursos?categoria=${encodeURIComponent(tab)}`)
  const act = useAction()
  const [mov, setMov] = useState<{ r: Row; tipo: string } | null>(null)
  const [min, setMin] = useState<Row | null>(null)
  const [nuevo, setNuevo] = useState(false)
  const [v, setV] = useState({ cantidad: 1, nota: '', stock_minimo: 0 })
  const [n, setN] = useState({ nombre: '', categoria_id: 1, stock: 0, stock_minimo: 0, tallas: '' })
  const esEpp = (cat.data?.categorias ?? []).find((c: Row) => c.id === n.categoria_id)?.nombre === 'EPP'
  const low = (q.data ?? []).filter((r: Row) => r.bajo_minimo)

  return (
    <Page title="Inventario" actions={<Btn className="btn btn-primary" onClick={() => setNuevo(true)}>Nuevo recurso</Btn>}>
      <Tabs value={tab} onChange={setTab} tabs={[['', 'Todos'], ...((cat.data?.categorias ?? []) as Row[]).map((c): [string, string] => [c.nombre, c.nombre])]} />
      {low.length > 0 && <Banner kind="warn"><b>{plural(low.length, 'recurso', 'recursos')} en o bajo el mínimo:</b> {low.map((r: Row) => r.nombre).join(', ')}.</Banner>}
      <Loading q={q} />
      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Recurso</th><th>Categoría</th><th>Stock</th><th>Mínimo</th><th></th></tr></thead>
          <tbody>
            {(q.data ?? []).map((r: Row) => (
              <tr key={r.id}>
                <td>{r.nombre}{r.en_espera > 0 && <div className="text-xs text-amber-700">{r.en_espera} solicitudes en espera</div>}</td>
                <td>{r.categoria}</td>
                <td className={r.bajo_minimo ? 'font-bold text-red-700' : ''}>{r.stock}</td>
                <td>{r.stock_minimo}</td>
                <td className="whitespace-nowrap text-right">
                  <Btn className="btn btn-sm" onClick={() => { setV({ ...v, cantidad: 1, nota: '' }); setMov({ r, tipo: 'ingreso' }) }}>Ingreso</Btn>{' '}
                  <Btn className="btn btn-sm" onClick={() => { setV({ ...v, cantidad: 1, nota: '' }); setMov({ r, tipo: 'salida' }) }}>Salida</Btn>{' '}
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
          footer={<Btn className="btn btn-primary" onClick={async () => { if (await act(`/recursos/${mov.r.id}/movimientos`, { tipo: mov.tipo, cantidad: v.cantidad, nota: v.nota }, 'Movimiento registrado.')) setMov(null) }}>Registrar</Btn>}>
          <p className="muted">Stock actual: {mov.r.stock}</p>
          <Field label="Cantidad"><input type="number" min={1} className="inp" value={v.cantidad} onChange={(e) => setV({ ...v, cantidad: +e.target.value })} /></Field>
          <Field label="Nota"><input className="inp" value={v.nota} onChange={(e) => setV({ ...v, nota: e.target.value })} /></Field>
        </Modal>
      )}
      {min && (
        <Modal title={`Stock mínimo: ${min.nombre}`} onClose={() => setMin(null)}
          footer={<Btn className="btn btn-primary" onClick={async () => { if (await act(`/recursos/${min.id}`, { stock_minimo: v.stock_minimo }, 'Mínimo actualizado.', 'PATCH')) setMin(null) }}>Guardar</Btn>}>
          <Field label="Mínimo"><input type="number" min={0} className="inp" value={v.stock_minimo} onChange={(e) => setV({ ...v, stock_minimo: +e.target.value })} /></Field>
        </Modal>
      )}
      {nuevo && (
        <Modal title="Nuevo recurso" onClose={() => setNuevo(false)}
          footer={<Btn className="btn btn-primary" onClick={async () => { if (await act('/recursos', { ...n, tallas: esEpp ? n.tallas : '' }, 'Recurso registrado.')) setNuevo(false) }}>Guardar</Btn>}>
          <Field label="Nombre"><input className="inp" value={n.nombre} onChange={(e) => setN({ ...n, nombre: e.target.value })} /></Field>
          <Field label="Categoría">
            <select className="inp" value={n.categoria_id} onChange={(e) => setN({ ...n, categoria_id: +e.target.value })}>
              {(cat.data?.categorias ?? []).map((c: Row) => <option key={c.id} value={c.id}>{c.nombre}</option>)}
            </select>
          </Field>
          {esEpp && (
            <Field label="Tallas que puede elegir el trabajador">
              <select className="inp" value={n.tallas} onChange={(e) => setN({ ...n, tallas: e.target.value })}>
                <option value="">Talla única</option>
                <option value="letras">Por letras (XS a XXL)</option>
                <option value="calzado">Calzado (35 a 45)</option>
              </select>
            </Field>
          )}
          <div className="grid grid-cols-2 gap-3">
            <Field label="Stock inicial"><input type="number" min={0} className="inp" value={n.stock} onChange={(e) => setN({ ...n, stock: +e.target.value })} /></Field>
            <Field label="Mínimo"><input type="number" min={0} className="inp" value={n.stock_minimo} onChange={(e) => setN({ ...n, stock_minimo: +e.target.value })} /></Field>
          </div>
        </Modal>
      )}
    </Page>
  )
}
