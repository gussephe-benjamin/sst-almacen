import { useState } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { Download, Plus, Trash2 } from 'lucide-react'
import type { Row } from '../api/client'
import { Banner, Chip, Field, Historial, Img, Loading, Modal, Page, PhotoField, fmtD, today, useAction, useApi } from '../components/ui'

export function Hallazgos() {
  const cat = useApi<Row>('/catalogos')
  const [sp, setSp] = useSearchParams()
  const qs = sp.toString()
  const q = useApi(`/hallazgos?${qs}`)
  const nav = useNavigate()
  const [nuevo, setNuevo] = useState(false)
  const areas: Row[] = cat.data?.areas ?? []
  const set = (k: string, v: string) => { const n = new URLSearchParams(sp); if (v) n.set(k, v); else n.delete(k); setSp(n) }

  return (
    <Page title="Hallazgos" sub="Registro de inspecciones y su seguimiento hasta el cierre."
      actions={<>
        <a className="btn" href={`/api/hallazgos/export.xlsx?${qs}`}><Download size={14} /> Excel</a>
        <a className="btn" href={`/api/hallazgos/export.pdf?${qs}`}><Download size={14} /> PDF</a>
        <button className="btn btn-primary" onClick={() => setNuevo(true)}><Plus size={14} /> Registrar hallazgo</button>
      </>}>
      <div className="card mb-3 grid gap-2 sm:grid-cols-5">
        <input className="inp" placeholder="Buscar…" defaultValue={sp.get('q') ?? ''} onKeyDown={(e) => e.key === 'Enter' && set('q', (e.target as HTMLInputElement).value)} />
        <select className="inp" value={sp.get('estado') ?? ''} onChange={(e) => set('estado', e.target.value)}>
          <option value="">Todos los estados</option>
          <option value="registrado">Registrado</option><option value="en_atencion">En atención</option>
          <option value="cerrado">Cerrado</option><option value="descartado">Descartado</option>
        </select>
        <select className="inp" value={sp.get('riesgo') ?? ''} onChange={(e) => set('riesgo', e.target.value)}>
          <option value="">Todo riesgo</option><option value="alto">Alto</option><option value="medio">Medio</option><option value="bajo">Bajo</option>
        </select>
        <select className="inp" value={sp.get('ubicacion') ?? ''} onChange={(e) => set('ubicacion', e.target.value)}>
          <option value="">Toda ubicación</option>
          {areas.filter((a) => a.es_ubicacion).map((a) => <option key={a.id} value={a.id}>{a.nombre}</option>)}
        </select>
        <select className="inp" value={sp.get('responsable') ?? ''} onChange={(e) => set('responsable', e.target.value)}>
          <option value="">Toda área responsable</option>
          {areas.filter((a) => a.puede_resolver_hallazgos).map((a) => <option key={a.id} value={a.id}>{a.nombre}</option>)}
        </select>
      </div>
      <Loading q={q} />
      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Código</th><th>Descripción</th><th>Ubicación</th><th>Área responsable</th><th>Riesgo</th><th>Tareas</th><th>Estado</th></tr></thead>
          <tbody>
            {(q.data ?? []).map((h: Row) => (
              <tr key={h.id} className="click" onClick={() => nav(`/hallazgos/${h.id}`)}>
                <td>{h.codigo}<div className="text-xs text-slate-500">{fmtD(h.created_at)}</div></td>
                <td>{h.descripcion}<div className="text-xs text-slate-500">{h.tipo}</div></td>
                <td>{h.ubicacion}</td><td>{h.area_responsable}</td><td><Chip s={h.nivel_riesgo} /></td>
                <td>{h.tareas ? `${h.tareas_cerradas}/${h.tareas}` : '—'}</td><td><Chip s={h.estado} /></td>
              </tr>
            ))}
          </tbody>
        </table>
        {q.data?.length === 0 && <p className="muted p-4">No hay hallazgos con esos filtros.</p>}
      </div>
      {nuevo && <NuevoHallazgo cat={cat.data} onClose={() => setNuevo(false)} />}
    </Page>
  )
}

function NuevoHallazgo({ cat, onClose }: { cat?: Row; onClose: () => void }) {
  const act = useAction()
  const nav = useNavigate()
  const [f, setF] = useState({ descripcion: '', detalle: '', tipo_id: 0, ubicacion_id: 0, area_responsable_id: 0, nivel_riesgo: 'medio', prioridad: 'media', fotos: [] as string[] })
  const [foto, setFoto] = useState('')
  const areas: Row[] = cat?.areas ?? []
  const save = async () => {
    const r = await act('/hallazgos', { ...f, fotos: foto ? [foto] : [] }, 'Hallazgo registrado.')
    if (r) nav(`/hallazgos/${r.id}`)
  }
  const sel = (k: string, opts: Row[]) => (
    <select className="inp" value={(f as Row)[k]} onChange={(e) => setF({ ...f, [k]: +e.target.value })}>
      <option value={0}>Selecciona…</option>
      {opts.map((o) => <option key={o.id} value={o.id}>{o.nombre}</option>)}
    </select>
  )
  return (
    <Modal title="Registrar hallazgo" onClose={onClose} footer={<><button className="btn" onClick={onClose}>Cancelar</button><button className="btn btn-primary" onClick={save}>Registrar</button></>}>
      <Field label="Descripción breve"><input className="inp" value={f.descripcion} onChange={(e) => setF({ ...f, descripcion: e.target.value })} /></Field>
      <Field label="Detalle"><textarea className="inp" rows={3} value={f.detalle} onChange={(e) => setF({ ...f, detalle: e.target.value })} /></Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Tipo">{sel('tipo_id', cat?.tipos ?? [])}</Field>
        <Field label="Ubicación">{sel('ubicacion_id', areas.filter((a) => a.es_ubicacion))}</Field>
        <Field label="Área responsable">{sel('area_responsable_id', areas.filter((a) => a.puede_resolver_hallazgos))}</Field>
        <Field label="Nivel de riesgo">
          <select className="inp" value={f.nivel_riesgo} onChange={(e) => setF({ ...f, nivel_riesgo: e.target.value })}><option value="alto">Alto</option><option value="medio">Medio</option><option value="bajo">Bajo</option></select>
        </Field>
        <Field label="Prioridad">
          <select className="inp" value={f.prioridad} onChange={(e) => setF({ ...f, prioridad: e.target.value })}><option value="alta">Alta</option><option value="media">Media</option><option value="baja">Baja</option></select>
        </Field>
      </div>
      <PhotoField value={foto} onChange={setFoto} />
    </Modal>
  )
}

export function HallazgoDetalle() {
  const { id } = useParams()
  const q = useApi<Row>(`/hallazgos/${id}`)
  const act = useAction()
  const [modal, setModal] = useState<'' | 'tarea' | 'descartar'>('')
  const [motivo, setMotivo] = useState('')
  const h = q.data
  const abierto = h && (h.estado === 'registrado' || h.estado === 'en_atencion')
  const directo = h && (h.area_responsable === 'SST' || h.area_responsable === 'Calidad')
  return (
    <Page title={h ? `${h.codigo} · ${h.descripcion}` : 'Hallazgo'} actions={h && <><Chip s={h.nivel_riesgo} /><Chip s={h.estado} /></>}>
      <Loading q={q} />
      {h && (
        <div className="grid gap-4 lg:grid-cols-3">
          <div className="space-y-4 lg:col-span-2">
            <div className="card space-y-3">
              {h.estado === 'descartado' && <Banner kind="info">Descartado: {h.motivo_descarte}</Banner>}
              <p className="text-sm">{h.detalle}</p>
              <dl className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-3">
                <div><dt className="text-xs text-slate-500">Tipo</dt><dd>{h.tipo}</dd></div>
                <div><dt className="text-xs text-slate-500">Ubicación</dt><dd>{h.ubicacion}</dd></div>
                <div><dt className="text-xs text-slate-500">Área responsable</dt><dd>{h.area_responsable}</dd></div>
                <div><dt className="text-xs text-slate-500">Prioridad</dt><dd><Chip s={h.prioridad} /></dd></div>
                <div><dt className="text-xs text-slate-500">Responsable SST</dt><dd>{h.responsable}</dd></div>
                <div><dt className="text-xs text-slate-500">Fecha</dt><dd>{fmtD(h.created_at)}</dd></div>
              </dl>
              <div className="flex flex-wrap gap-2">{(h.fotos ?? []).map((f: Row) => <Img key={f.id} id={f.id} className="h-32 w-44" />)}</div>
              {abierto && (
                <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-3">
                  {!directo && <button className="btn btn-primary" onClick={() => setModal('tarea')}><Plus size={14} /> Crear tarea</button>}
                  {(directo ? h.tareas === 0 : h.tareas > 0) && <button className="btn btn-primary" onClick={() => act(`/hallazgos/${id}/cerrar`, {}, 'Hallazgo cerrado.')}>Cerrar hallazgo</button>}
                  <button className="btn btn-danger" onClick={() => setModal('descartar')}>Descartar</button>
                </div>
              )}
              {abierto && !directo && h.tareas > h.tareas_cerradas && <p className="muted">El hallazgo se podrá cerrar cuando todas sus tareas estén cerradas.</p>}
            </div>
            <div className="card">
              <h2 className="mb-2 font-semibold">Tareas</h2>
              {h.tareas_lista?.length ? (
                <table className="tbl"><tbody>
                  {h.tareas_lista.map((t: Row) => (
                    <tr key={t.id}><td><Link className="text-blue-700" to={`/tareas/${t.id}`}>{t.codigo}</Link></td><td>{t.titulo}<div className="text-xs text-slate-500">{t.responsable} · vence {fmtD(t.fecha_limite)}</div></td><td>{t.pasos_hechos}/{t.pasos_total}</td><td><Chip s={t.estado} /></td></tr>
                  ))}
                </tbody></table>
              ) : <p className="muted">{directo ? 'SST o Calidad pueden cerrar este hallazgo directamente.' : 'Aún no hay tareas.'}</p>}
            </div>
          </div>
          <div className="card"><h2 className="mb-2 font-semibold">Historial</h2><Historial items={h.historial} /></div>
        </div>
      )}
      {modal === 'tarea' && h && <NuevaTarea h={h} onClose={() => setModal('')} />}
      {modal === 'descartar' && (
        <Modal title="Descartar hallazgo" onClose={() => setModal('')}
          footer={<button className="btn btn-danger" onClick={async () => { if (await act(`/hallazgos/${id}/descartar`, { motivo }, 'Hallazgo descartado.')) setModal('') }}>Descartar</button>}>
          <Field label="Motivo"><textarea className="inp" rows={3} value={motivo} onChange={(e) => setMotivo(e.target.value)} /></Field>
        </Modal>
      )}
    </Page>
  )
}

function NuevaTarea({ h, onClose }: { h: Row; onClose: () => void }) {
  const cat = useApi<Row>('/catalogos')
  const act = useAction()
  const rol = h.area_responsable === 'Mantenimiento' ? 'mantenimiento' : 'administracion'
  const puestos: Row[] = (cat.data?.puestos_apoyo ?? []).filter((p: Row) => p.rol === rol)
  const [f, setF] = useState({ titulo: '', indicaciones: '', puesto_id: 0, fecha_limite: today(7), pasos: [''] })
  const puesto = f.puesto_id || puestos[0]?.id || 0
  const save = async () => { if (await act(`/hallazgos/${h.id}/tareas`, { ...f, puesto_id: puesto }, 'Tarea creada y asignada.')) onClose() }
  return (
    <Modal title={`Nueva tarea para ${h.area_responsable}`} onClose={onClose} footer={<><button className="btn" onClick={onClose}>Cancelar</button><button className="btn btn-primary" onClick={save}>Crear tarea</button></>}>
      <Field label="Título"><input className="inp" value={f.titulo} onChange={(e) => setF({ ...f, titulo: e.target.value })} /></Field>
      <Field label="Indicaciones"><textarea className="inp" rows={3} value={f.indicaciones} onChange={(e) => setF({ ...f, indicaciones: e.target.value })} /></Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Responsable">
          <select className="inp" value={puesto} onChange={(e) => setF({ ...f, puesto_id: +e.target.value })}>
            {puestos.map((p) => <option key={p.id} value={p.id}>{p.persona ?? 'Sin asignar'} · {p.nombre}</option>)}
          </select>
        </Field>
        <Field label="Fecha límite"><input type="date" className="inp" value={f.fecha_limite} onChange={(e) => setF({ ...f, fecha_limite: e.target.value })} /></Field>
      </div>
      <div>
        <span className="lbl">Pasos</span>
        {f.pasos.map((p, i) => (
          <div key={i} className="mb-2 flex gap-2">
            <input className="inp" placeholder={`Paso ${i + 1}`} value={p} onChange={(e) => setF({ ...f, pasos: f.pasos.map((x, j) => (j === i ? e.target.value : x)) })} />
            {f.pasos.length > 1 && <button className="btn" onClick={() => setF({ ...f, pasos: f.pasos.filter((_, j) => j !== i) })}><Trash2 size={14} /></button>}
          </div>
        ))}
        <button className="btn btn-sm" onClick={() => setF({ ...f, pasos: [...f.pasos, ''] })}><Plus size={12} /> Agregar paso</button>
      </div>
    </Modal>
  )
}
