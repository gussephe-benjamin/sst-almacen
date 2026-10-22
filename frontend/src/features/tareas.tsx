import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { CheckCircle2, Circle, RotateCcw } from 'lucide-react'
import type { Me, Row } from '../api/client'
import { Btn, confirmar, Banner, Chip, Field, Historial, Img, Loading, Modal, Page, PhotoField, Tabs, fmtD, fmtDT, today, useAction, useApi } from '../components/ui'

const GRUPOS: Record<string, string[]> = {
  por_hacer: ['asignada', 'modificacion', 'devuelta'],
  en_proceso: ['en_proceso'],
  terminadas: ['por_validar', 'cerrada'],
}

export function Tareas({ me }: { me: Me }) {
  const q = useApi('/tareas')
  const nav = useNavigate()
  const [tab, setTab] = useState(me.rol === 'sst' ? 'atencion' : 'por_hacer')
  const all: Row[] = q.data ?? []
  const vencida = (t: Row) => t.estado !== 'cerrada' && t.fecha_limite < today()

  if (me.rol !== 'sst') {
    return (
      <Page title="Mis tareas" sub="Tareas asignadas a tu puesto por SST.">
        <Loading q={q} />
        <div className="grid gap-4 md:grid-cols-3">
          {Object.entries({ por_hacer: 'Por hacer', en_proceso: 'En proceso', terminadas: 'Terminadas' }).map(([g, l]) => (
            <div key={g}>
              <h2 className="mb-2 text-sm font-semibold text-slate-600">{l} ({all.filter((t) => GRUPOS[g].includes(t.estado)).length})</h2>
              <div className="space-y-2">
                {all.filter((t) => GRUPOS[g].includes(t.estado)).map((t) => (
                  <Link key={t.id} to={`/tareas/${t.id}`} className="card block hover:border-blue-300">
                    <div className="flex justify-between text-xs text-slate-500"><span>{t.codigo} · {t.hallazgo_codigo}</span><Chip s={t.estado} /></div>
                    <div className="mt-1 font-medium">{t.titulo}</div>
                    <div className={`mt-1 text-xs ${vencida(t) ? 'font-semibold text-red-700' : 'text-slate-500'}`}>Vence {fmtD(t.fecha_limite)} · {t.pasos_hechos}/{t.pasos_total} pasos</div>
                  </Link>
                ))}
              </div>
            </div>
          ))}
        </div>
      </Page>
    )
  }

  const filtros: Record<string, (t: Row) => boolean> = {
    atencion: (t) => t.estado === 'por_validar' || t.estado === 'modificacion',
    abiertas: (t) => t.estado !== 'cerrada',
    cerradas: (t) => t.estado === 'cerrada',
    todas: () => true,
  }
  const rows = all.filter(filtros[tab])
  return (
    <Page title="Tareas a áreas">
      <Tabs value={tab} onChange={setTab} tabs={[['atencion', `Requieren tu atención (${all.filter(filtros.atencion).length})`], ['abiertas', 'Abiertas'], ['cerradas', 'Cerradas'], ['todas', 'Todas']]} />
      <Loading q={q} />
      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Código</th><th>Tarea</th><th>Responsable</th><th>Vence</th><th>Pasos</th><th>Estado</th></tr></thead>
          <tbody>
            {rows.map((t) => (
              <tr key={t.id} className="click" onClick={() => nav(`/tareas/${t.id}`)}>
                <td>{t.codigo}<div className="text-xs text-slate-500">{t.hallazgo_codigo}</div></td><td>{t.titulo}</td>
                <td>{t.responsable}<div className="text-xs text-slate-500">{t.area}</div></td>
                <td className={vencida(t) ? 'font-semibold text-red-700' : ''}>{fmtD(t.fecha_limite)}</td>
                <td>{t.pasos_hechos}/{t.pasos_total}</td><td><Chip s={t.estado} /></td>
              </tr>
            ))}
          </tbody>
        </table>
        {rows.length === 0 && <p className="muted p-4">No hay tareas aquí.</p>}
      </div>
    </Page>
  )
}

export function TareaDetalle({ me }: { me: Me }) {
  const { id } = useParams()
  const q = useApi<Row>(`/tareas/${id}`)
  const act = useAction()
  const [modal, setModal] = useState<'' | 'mod' | 'rechazarMod' | 'devolver' | number>('')
  const [v, setV] = useState({ tipo: 'plazo', nueva_fecha: today(7), motivo: '', respuesta: '', comentario: '', evidencia_id: '', pasos: [] as number[] })
  const t = q.data
  const sst = me.rol === 'sst'
  const resp = !sst
  const editable = resp && t && (t.estado === 'en_proceso' || t.estado === 'devuelta')
  const mod: Row | undefined = t?.modificaciones?.find((m: Row) => m.estado === 'pendiente')
  const run = async (path: string, body: unknown, msg: string) => { if (await act(`/tareas/${id}${path}`, body, msg)) setModal('') }

  return (
    <Page back="/tareas" title={t ? `${t.codigo} · ${t.titulo}` : 'Tarea'} actions={t && <Chip s={t.estado} />}>
      <Loading q={q} />
      {t && (
        <div className="grid gap-4 lg:grid-cols-3">
          <div className="space-y-4 lg:col-span-2">
            <div className="card space-y-3">
              {t.estado === 'devuelta' && <Banner kind="err"><b>SST devolvió la tarea:</b> {t.devolucion_comentario}</Banner>}
              {t.estado === 'por_validar' && <Banner kind="info">{sst ? 'El responsable terminó la tarea. Revisa la evidencia para cerrarla o devolverla.' : 'Enviada a SST para validación.'}</Banner>}
              {mod && <Banner kind="warn"><b>Modificación de {mod.tipo} solicitada</b>{mod.nueva_fecha && <> · nueva fecha {fmtD(mod.nueva_fecha)}</>}<div>{mod.motivo}</div></Banner>}
              <p className="text-sm">{t.indicaciones}</p>
              <dl className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
                <div><dt className="text-xs text-slate-500">Hallazgo</dt><dd>{sst ? <Link className="text-blue-700" to={`/hallazgos/${t.hallazgo_id}`}>{t.hallazgo_codigo}</Link> : t.hallazgo_codigo}</dd></div>
                <div><dt className="text-xs text-slate-500">Ubicación</dt><dd>{t.ubicacion}</dd></div>
                <div><dt className="text-xs text-slate-500">Responsable</dt><dd>{t.responsable}</dd></div>
                <div><dt className="text-xs text-slate-500">Fecha límite</dt><dd>{fmtD(t.fecha_limite)}</dd></div>
              </dl>
              <div className="flex flex-wrap gap-2 border-t border-slate-100 pt-3">
                {resp && t.estado === 'asignada' && <>
                  <Btn className="btn btn-primary" onClick={() => run('/aceptar', {}, 'Tarea aceptada.')}>Aceptar tarea</Btn>
                  <Btn className="btn" onClick={() => setModal('mod')}>Pedir modificación</Btn>
                </>}
                {editable && <Btn className="btn btn-primary" onClick={() => confirmar('¿Enviar la tarea a SST para validación?') ? run('/terminar', {}, 'Tarea enviada a SST para validación.') : undefined}>Terminar y enviar a SST</Btn>}
                {sst && t.estado === 'modificacion' && <>
                  <Btn className="btn btn-primary" onClick={() => run('/modificacion/aprobar', {}, 'Modificación aprobada.')}>Aprobar modificación</Btn>
                  <Btn className="btn btn-danger" onClick={() => setModal('rechazarMod')}>Rechazar</Btn>
                </>}
                {sst && t.estado === 'por_validar' && <>
                  <Btn className="btn btn-primary" onClick={() => confirmar('¿Validar y cerrar la tarea? Esta acción no se puede deshacer.') ? run('/cerrar', {}, 'Tarea validada y cerrada.') : undefined}>Validar y cerrar</Btn>
                  <Btn className="btn btn-danger" onClick={() => { setV({ ...v, comentario: '', pasos: [] }); setModal('devolver') }}>Devolver</Btn>
                </>}
              </div>
            </div>
            <div className="card">
              <h2 className="mb-2 font-semibold">Pasos</h2>
              <ol className="space-y-3">
                {(t.pasos ?? []).map((p: Row) => (
                  <li key={p.orden} className={`rounded-md border p-3 ${p.rehacer ? 'border-red-300 bg-red-50' : 'border-slate-200'}`}>
                    <div className="flex items-center gap-2">
                      {p.completado ? <CheckCircle2 size={18} className="text-emerald-600" /> : <Circle size={18} className="text-slate-400" />}
                      <span className="flex-1 font-medium">{p.orden}. {p.titulo}</span>
                      {p.rehacer && !p.completado && <Chip s="devuelta">Rehacer</Chip>}
                      {editable && !p.completado && <Btn className="btn btn-sm btn-primary" onClick={() => { setV({ ...v, comentario: '', evidencia_id: '' }); setModal(p.orden) }}>Completar</Btn>}
                      {editable && p.completado && <Btn className="btn btn-sm" onClick={() => run(`/pasos/${p.orden}/reabrir`, {}, 'Paso reabierto.')}><RotateCcw size={12} /> Reabrir</Btn>}
                    </div>
                    {p.evidencia_id && (
                      <div className="mt-2 flex gap-3">
                        <Img id={p.evidencia_id} className="h-20 w-28" />
                        <div className="text-sm">{p.comentario}<div className="text-xs text-slate-500">{p.completado_por} · {fmtDT(p.completado_at)}</div></div>
                      </div>
                    )}
                  </li>
                ))}
              </ol>
            </div>
          </div>
          <div className="card"><h2 className="mb-2 font-semibold">Historial</h2><Historial items={t.historial} /></div>
        </div>
      )}

      {modal === 'mod' && (
        <Modal title="Pedir modificación" onClose={() => setModal('')} footer={<Btn className="btn btn-primary" onClick={() => run('/modificacion', v, 'Solicitud enviada a SST.')}>Enviar</Btn>}>
          <Field label="Tipo">
            <select className="inp" value={v.tipo} onChange={(e) => setV({ ...v, tipo: e.target.value })}><option value="plazo">Cambio de plazo</option><option value="alcance">Cambio de alcance</option></select>
          </Field>
          {v.tipo === 'plazo' && <Field label="Nueva fecha propuesta"><input type="date" className="inp" value={v.nueva_fecha} onChange={(e) => setV({ ...v, nueva_fecha: e.target.value })} /></Field>}
          <Field label="Motivo"><textarea className="inp" rows={3} value={v.motivo} onChange={(e) => setV({ ...v, motivo: e.target.value })} /></Field>
        </Modal>
      )}
      {modal === 'rechazarMod' && (
        <Modal title="Rechazar modificación" onClose={() => setModal('')} footer={<Btn className="btn btn-danger" onClick={() => run('/modificacion/rechazar', { respuesta: v.respuesta }, 'Modificación rechazada.')}>Rechazar</Btn>}>
          <Field label="Respuesta para el responsable"><textarea className="inp" rows={3} value={v.respuesta} onChange={(e) => setV({ ...v, respuesta: e.target.value })} /></Field>
        </Modal>
      )}
      {modal === 'devolver' && t && (
        <Modal title="Devolver tarea" onClose={() => setModal('')} footer={<Btn className="btn btn-danger" onClick={() => run('/devolver', { comentario: v.comentario, pasos: v.pasos }, 'Tarea devuelta.')}>Devolver</Btn>}>
          <Field label="Comentario"><textarea className="inp" rows={3} value={v.comentario} onChange={(e) => setV({ ...v, comentario: e.target.value })} /></Field>
          <span className="lbl">Pasos a rehacer</span>
          {(t.pasos ?? []).map((p: Row) => (
            <label key={p.orden} className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={v.pasos.includes(p.orden)} onChange={(e) => setV({ ...v, pasos: e.target.checked ? [...v.pasos, p.orden] : v.pasos.filter((x) => x !== p.orden) })} />
              {p.orden}. {p.titulo}
            </label>
          ))}
        </Modal>
      )}
      {typeof modal === 'number' && (
        <Modal title={`Completar paso ${modal}`} onClose={() => setModal('')}
          footer={<Btn className="btn btn-primary" onClick={() => run(`/pasos/${modal}/completar`, { evidencia_id: v.evidencia_id, comentario: v.comentario }, 'Paso completado.')}>Guardar</Btn>}>
          <PhotoField label="Foto de evidencia" value={v.evidencia_id} onChange={(x) => setV({ ...v, evidencia_id: x })} />
          <Field label="Comentario"><textarea className="inp" rows={3} value={v.comentario} onChange={(e) => setV({ ...v, comentario: e.target.value })} /></Field>
        </Modal>
      )}
    </Page>
  )
}
