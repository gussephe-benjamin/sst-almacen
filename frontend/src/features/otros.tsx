import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import type { Me, Row } from '../api/client'
import { Btn, Banner, Chip, Field, Img, Loading, Modal, Page, PhotoField, Tabs, fmtD, fmtDT, label, today, useAction, useApi } from '../components/ui'

// ---------- Dashboard ----------

function Bars({ rows, onClick }: { rows: Row[]; onClick?: (r: Row) => void }) {
  const max = Math.max(1, ...rows.map((r) => r.cantidad))
  return (
    <div className="space-y-2">
      {rows.map((r) => (
        <div key={r.nombre} className={`text-sm ${onClick ? 'cursor-pointer hover:opacity-80' : ''}`} onClick={() => onClick?.(r)}>
          <div className="flex justify-between"><span>{label(r.nombre)}</span><span className="text-slate-500">{r.cantidad}{r.abiertos !== undefined && ` · ${r.abiertos} abiertos · ${r.porcentaje}%`}</span></div>
          <div className="h-2 rounded bg-slate-100"><div className="h-2 rounded bg-blue-600" style={{ width: `${(100 * r.cantidad) / max}%` }} /></div>
        </div>
      ))}
    </div>
  )
}

export function Dashboard() {
  const q = useApi<Row>('/dashboard')
  const nav = useNavigate()
  const d = q.data
  const kpi = (l: string, v: unknown, to?: string) => (
    <div className={`card ${to ? 'cursor-pointer hover:border-blue-300' : ''}`} onClick={() => to && nav(to)}>
      <div className="text-xs text-slate-500">{l}</div>
      <div className="text-2xl font-bold">{(v as string) ?? '—'}</div>
    </div>
  )
  const maxMes = Math.max(1, ...(d?.por_mes ?? []).map((m: Row) => m.alto + m.medio + m.bajo))
  return (
    <Page title="Dashboard" sub="Resumen de la gestión de SST del almacén.">
      <Loading q={q} />
      {d && <>
        <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-6">
          {kpi('Hallazgos abiertos', d.abiertos, '/hallazgos')}
          {kpi('Abiertos de riesgo alto', d.abiertos_alto, '/hallazgos?riesgo=alto')}
          {kpi('Días promedio de atención', d.dias_promedio)}
          {kpi('Solicitudes de EPP nuevas', d.epp_nuevas, '/epp')}
          {kpi('Tareas por validar', d.tareas_por_validar, '/tareas')}
          {kpi('Recursos bajo mínimo', d.bajo_minimo.length, '/inventario')}
        </div>
        <div className="mb-4 card">
          <h2 className="mb-2 font-semibold">Pendientes</h2>
          <ul className="space-y-1 text-sm">
            {d.epp_nuevas > 0 && <li><Link className="text-blue-700" to="/epp">{d.epp_nuevas} solicitudes de EPP por revisar</Link></li>}
            {d.tareas_por_validar > 0 && <li><Link className="text-blue-700" to="/tareas">{d.tareas_por_validar} tareas por validar</Link></li>}
            {d.modificaciones > 0 && <li><Link className="text-blue-700" to="/tareas">{d.modificaciones} solicitudes de modificación</Link></li>}
            {d.bajo_minimo.length > 0 && <li><Link className="text-blue-700" to="/inventario">{d.bajo_minimo.length} recursos en o bajo el mínimo: {d.bajo_minimo.slice(0, 3).map((r: Row) => r.nombre).join(', ')}</Link></li>}
          </ul>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <div className="card"><h2 className="mb-3 font-semibold">Por área responsable</h2><Bars rows={d.por_area} onClick={(r) => nav(`/hallazgos?responsable=${r.id}`)} /></div>
          <div className="card"><h2 className="mb-3 font-semibold">Por tipo</h2><Bars rows={d.por_tipo} /></div>
          <div className="card">
            <h2 className="mb-3 font-semibold">Por nivel de riesgo</h2><Bars rows={d.por_riesgo} />
            <p className="muted mt-3">Días promedio de atención: {d.por_riesgo.map((r: Row) => `${label(r.nombre)} ${r.dias_promedio ?? '—'}`).join(' · ')}</p>
          </div>
          <div className="card"><h2 className="mb-3 font-semibold">Por ubicación</h2><Bars rows={d.por_ubicacion} /></div>
          <div className="card md:col-span-2">
            <h2 className="mb-3 font-semibold">Por mes (últimos 6)</h2>
            <div className="flex h-40 items-end gap-3">
              {d.por_mes.map((m: Row) => (
                <div key={m.mes} className="flex flex-1 flex-col items-center">
                  <div className="flex w-full flex-col-reverse" style={{ height: `${(120 * (m.alto + m.medio + m.bajo)) / maxMes}px` }}>
                    <div className="bg-red-500" style={{ flex: m.alto }} /><div className="bg-amber-400" style={{ flex: m.medio }} /><div className="bg-emerald-500" style={{ flex: m.bajo }} />
                  </div>
                  <span className="mt-1 text-xs text-slate-500">{m.mes}</span>
                </div>
              ))}
            </div>
            <p className="mt-2 text-xs text-slate-500"><span className="text-red-600">■</span> Alto <span className="text-amber-500">■</span> Medio <span className="text-emerald-600">■</span> Bajo</p>
          </div>
        </div>
      </>}
    </Page>
  )
}

// ---------- Biblioteca ----------

const TIPOS = [['afiche', 'Afiches'], ['flyer', 'Flyers'], ['procedimiento', 'Procedimientos'], ['instructivo', 'Instructivos']]

export function Biblioteca({ me }: { me: Me }) {
  const [tab, setTab] = useState('')
  const q = useApi(`/biblioteca?tipo=${tab}`)
  const act = useAction()
  const [ver, setVer] = useState<Row | null>(null)
  const [edit, setEdit] = useState<Row | null>(null)
  const sst = me.rol === 'sst'
  return (
    <Page title="Biblioteca" sub="Afiches, procedimientos e instructivos de seguridad."
      actions={sst && <Btn className="btn btn-primary" onClick={() => setEdit({ tipo: 'afiche', codigo: '', titulo: '', resumen: '', cuerpo: '', imagen_id: '' })}>Publicar</Btn>}>
      <Tabs value={tab} onChange={setTab} tabs={[['', 'Todo'], ...(TIPOS as [string, string][])]} />
      <Loading q={q} />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {(q.data ?? []).map((b: Row) => (
          <div key={b.id} className="card cursor-pointer hover:border-blue-300" onClick={() => setVer(b)}>
            <div className="text-xs uppercase text-slate-500">{b.tipo}{b.codigo && ` · ${b.codigo}`}</div>
            <div className="font-semibold">{b.titulo}</div>
            <p className="muted mt-1">{b.resumen}</p>
          </div>
        ))}
      </div>
      {ver && (
        <Modal title={ver.titulo} onClose={() => setVer(null)}
          footer={sst && <>
            <Btn className="btn btn-danger" onClick={async () => { if (confirm('¿Eliminar este contenido?') && await act(`/biblioteca/${ver.id}`, undefined, 'Contenido eliminado.', 'DELETE')) setVer(null) }}>Eliminar</Btn>
            <Btn className="btn" onClick={() => { setEdit({ ...ver, imagen_id: ver.imagen_id ?? '' }); setVer(null) }}>Editar</Btn>
          </>}>
          <div className="text-xs uppercase text-slate-500">{ver.tipo}{ver.codigo && ` · ${ver.codigo}`} · {fmtD(ver.created_at)}</div>
          <Img id={ver.imagen_id} className="w-full" />
          <p className="font-medium">{ver.resumen}</p>
          {String(ver.cuerpo).split(/\n\s*\n/).filter(Boolean).map((p, i) => <p key={i} className="text-sm">{p}</p>)}
        </Modal>
      )}
      {edit && (
        <Modal title={edit.id ? 'Editar contenido' : 'Publicar contenido'} onClose={() => setEdit(null)}
          footer={<Btn className="btn btn-primary" onClick={async () => {
            if (await act(edit.id ? `/biblioteca/${edit.id}` : '/biblioteca', edit, edit.id ? 'Contenido actualizado.' : 'Contenido publicado.', edit.id ? 'PUT' : 'POST')) setEdit(null)
          }}>Guardar</Btn>}>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Tipo"><select className="inp" value={edit.tipo} onChange={(e) => setEdit({ ...edit, tipo: e.target.value })}>{TIPOS.map(([v, l]) => <option key={v} value={v}>{l.slice(0, -1)}</option>)}</select></Field>
            <Field label="Código (opcional)"><input className="inp" value={edit.codigo ?? ''} onChange={(e) => setEdit({ ...edit, codigo: e.target.value })} /></Field>
          </div>
          <Field label="Título"><input className="inp" value={edit.titulo} onChange={(e) => setEdit({ ...edit, titulo: e.target.value })} /></Field>
          <Field label="Resumen"><input className="inp" value={edit.resumen} onChange={(e) => setEdit({ ...edit, resumen: e.target.value })} /></Field>
          <Field label="Contenido (separa párrafos con una línea en blanco)"><textarea className="inp" rows={5} value={edit.cuerpo} onChange={(e) => setEdit({ ...edit, cuerpo: e.target.value })} /></Field>
          <PhotoField label="Imagen (opcional)" value={edit.imagen_id ?? ''} onChange={(id) => setEdit({ ...edit, imagen_id: id })} />
        </Modal>
      )}
    </Page>
  )
}

// ---------- Tareas internas ----------

export function Internas() {
  const q = useApi('/tareas-internas')
  const act = useAction()
  const [nueva, setNueva] = useState(false)
  const [f, setF] = useState({ titulo: '', responsable: '', fecha: today(7) })
  const cols: [string, string, string][] = [['pendiente', 'Pendientes', 'en_curso'], ['en_curso', 'En curso', 'completada'], ['completada', 'Completadas', 'pendiente']]
  return (
    <Page title="Tareas internas de SST" actions={<Btn className="btn btn-primary" onClick={() => setNueva(true)}>Nueva tarea</Btn>}>
      <Loading q={q} />
      <div className="grid gap-4 md:grid-cols-3">
        {cols.map(([est, l, next]) => (
          <div key={est}>
            <h2 className="mb-2 text-sm font-semibold text-slate-600">{l}</h2>
            <div className="space-y-2">
              {(q.data ?? []).filter((t: Row) => t.estado === est).map((t: Row) => (
                <div key={t.id} className="card">
                  <div className="font-medium">{t.titulo}</div>
                  <div className="text-xs text-slate-500">{t.responsable} · {fmtD(t.fecha)}</div>
                  <Btn className="btn btn-sm mt-2" onClick={() => act(`/tareas-internas/${t.id}`, { estado: next }, undefined, 'PATCH')}>
                    {est === 'completada' ? 'Reabrir' : `Mover a ${label(next).toLowerCase()}`}
                  </Btn>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>
      {nueva && (
        <Modal title="Nueva tarea interna" onClose={() => setNueva(false)}
          footer={<Btn className="btn btn-primary" onClick={async () => { if (await act('/tareas-internas', f, 'Tarea creada.')) setNueva(false) }}>Crear</Btn>}>
          <Field label="Título"><input className="inp" value={f.titulo} onChange={(e) => setF({ ...f, titulo: e.target.value })} /></Field>
          <Field label="Responsable"><input className="inp" placeholder="Por defecto, tú" value={f.responsable} onChange={(e) => setF({ ...f, responsable: e.target.value })} /></Field>
          <Field label="Fecha"><input type="date" className="inp" value={f.fecha} onChange={(e) => setF({ ...f, fecha: e.target.value })} /></Field>
        </Modal>
      )}
    </Page>
  )
}

// ---------- Puestos ----------

export function Puestos() {
  const q = useApi('/puestos')
  const pend = useApi('/usuarios/pendientes')
  const act = useAction()
  const [asig, setAsig] = useState<{ puesto_id: number; usuario_id: number } | null>(null)
  const [hist, setHist] = useState<Row | null>(null)
  const h = useApi<Row>(hist ? `/puestos/${hist.id}/historial` : null)
  const pendientes: Row[] = pend.data ?? []
  const rolChip = (r: string) => <Chip>{{ trabajador: 'Trabajador', sst: 'SST', mantenimiento: 'Mantenimiento', administracion: 'Administración' }[r]}</Chip>
  return (
    <Page title="Puestos y usuarios" sub="Las cuentas pertenecen al puesto, no a la persona. Cuando cambia el personal, el reemplazo ingresa con su DNI a la misma cuenta y el historial del puesto se conserva.">
      {pendientes.length > 0 && (
        <div className="card mb-4">
          <h2 className="mb-2 font-semibold">Registros por asignar <Chip s="en_espera">{pendientes.length}</Chip></h2>
          <table className="tbl"><tbody>
            {pendientes.map((p) => (
              <tr key={p.id}>
                <td><b>{p.nombres}</b><div className="text-xs text-slate-500">DNI {p.dni}</div></td>
                <td className="text-xs">{p.correo}<div className="text-slate-500">{p.celular}</div></td>
                <td className="text-xs">{p.area ?? '—'}</td>
                <td className="text-xs text-slate-500">Registrado {fmtDT(p.created_at)}</td>
                <td className="text-right"><Btn className="btn btn-sm btn-primary" onClick={() => setAsig({ puesto_id: 0, usuario_id: p.id })}>Asignar a un puesto</Btn></td>
              </tr>
            ))}
          </tbody></table>
        </div>
      )}
      <Loading q={q} />
      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Puesto</th><th>Rol</th><th>Persona actual</th><th>Celular</th><th>Desde</th><th></th></tr></thead>
          <tbody>
            {(q.data ?? []).map((p: Row) => (
              <tr key={p.id}>
                <td><b>{p.nombre}</b><div className="text-xs text-slate-500">{p.area}</div></td>
                <td>{rolChip(p.rol)}</td>
                <td>{p.persona ?? <span className="text-slate-400">Vacante</span>}{p.dni && <div className="text-xs text-slate-500">DNI {p.dni}</div>}</td>
                <td className="text-xs">{p.celular}</td><td className="text-xs">{fmtD(p.desde)}</td>
                <td className="whitespace-nowrap text-right">
                  <Btn className="btn btn-sm" onClick={() => setAsig({ puesto_id: p.id, usuario_id: pendientes[0]?.id ?? 0 })}>Cambiar personal</Btn>{' '}
                  <Btn className="btn btn-sm" onClick={() => setHist(p)}>Historial ({p.anteriores})</Btn>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {asig && (
        <Modal title="Asignar persona a un puesto" onClose={() => setAsig(null)}
          footer={<Btn className="btn btn-primary" onClick={async () => { if (await act(`/puestos/${asig.puesto_id}/asignar`, { usuario_id: asig.usuario_id }, 'Persona asignada al puesto.')) setAsig(null) }}>Asignar</Btn>}>
          {pendientes.length === 0 ? <Banner kind="warn">No hay registros pendientes. La persona debe registrarse primero con su DNI.</Banner> : <>
            <Field label="Persona">
              <select className="inp" value={asig.usuario_id} onChange={(e) => setAsig({ ...asig, usuario_id: +e.target.value })}>
                {pendientes.map((p) => <option key={p.id} value={p.id}>{p.nombres}, DNI {p.dni}</option>)}
              </select>
            </Field>
            <Field label="Puesto">
              <select className="inp" value={asig.puesto_id} onChange={(e) => setAsig({ ...asig, puesto_id: +e.target.value })}>
                <option value={0}>Selecciona…</option>
                {(q.data ?? []).map((p: Row) => <option key={p.id} value={p.id}>{p.nombre}{p.persona ? ` (hoy: ${p.persona})` : ''}</option>)}
              </select>
            </Field>
            <p className="muted">Si el puesto está ocupado, la persona actual sale del puesto. La nueva hereda sus solicitudes, tareas y avisos.</p>
          </>}
        </Modal>
      )}
      {hist && (
        <Modal title={`Historial: ${hist.nombre}`} onClose={() => setHist(null)}>
          <h3 className="font-semibold">Personas</h3>
          <ul className="text-sm">{(h.data?.asignaciones ?? []).map((a: Row, i: number) => <li key={i}>{a.persona} (DNI {a.dni}): {fmtD(a.desde)} – {a.hasta ? fmtD(a.hasta) : 'actual'}</li>)}</ul>
          <h3 className="font-semibold">Acciones del puesto</h3>
          <ul className="max-h-64 space-y-1 overflow-auto text-sm">{(h.data?.acciones ?? []).map((a: Row, i: number) => <li key={i}>{a.accion} <span className="text-xs text-slate-500">· {a.quien} · {fmtDT(a.fecha)}</span></li>)}</ul>
        </Modal>
      )}
    </Page>
  )
}

// ---------- Notificaciones ----------

const RUTA: Record<string, (id: number) => string> = {
  epp: (id) => `/epp/${id}`, tarea: (id) => `/tareas/${id}`, inventario: () => '/inventario', puestos: () => '/puestos', cuentas: () => '/cuentas', biblioteca: () => '/biblioteca',
}

export function Notificaciones() {
  const q = useApi('/notificaciones')
  const act = useAction()
  const nav = useNavigate()
  return (
    <Page title="Notificaciones" actions={<Btn className="btn" onClick={() => act('/notificaciones/leer-todas')}>Marcar todas como leídas</Btn>}>
      <Loading q={q} />
      {q.data?.length === 0 && <p className="muted">No tienes notificaciones.</p>}
      <div className="space-y-2">
        {(q.data ?? []).map((n: Row) => (
          <div key={n.id} className={`card flex cursor-pointer items-start gap-3 hover:border-blue-300 ${n.leida ? 'opacity-60' : ''}`}
            onClick={async () => { await act(`/notificaciones/${n.id}/leer`); const r = RUTA[n.ref_tipo]; if (r) nav(r(n.ref_id)) }}>
            {!n.leida && <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-blue-600" />}
            <div><div className="text-sm">{n.texto}</div><div className="text-xs text-slate-500">{fmtDT(n.created_at)}</div></div>
          </div>
        ))}
      </div>
    </Page>
  )
}
