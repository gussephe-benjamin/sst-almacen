import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import type { Me, Row } from '../api/client'
import { Btn, Banner, Chip, Field, Historial, Img, Loading, Modal, Page, PhotoField, Tabs, fmtD, fmtDT, today, useAction, useApi } from '../components/ui'

const MOTIVOS = ['Desgaste por uso', 'Rotura o daño', 'Pérdida', 'Talla incorrecta', 'Otro']

export function SolicitarEpp() {
  const cat = useApi<Row>('/catalogos')
  const act = useAction()
  const nav = useNavigate()
  const [f, setF] = useState({ recurso_id: 0, motivo: '', descripcion: '', foto_id: '' })
  const submit = async () => {
    const r = await act('/epp', f, 'Solicitud enviada a SST.')
    if (r) nav('/mis-solicitudes')
  }
  const listo = f.recurso_id && f.motivo && f.descripcion.trim() && f.foto_id
  return (
    <Page title="Solicitar cambio de EPP" sub="Adjunta una foto del EPP dañado. SST revisará tu solicitud.">
      <div className="card max-w-md space-y-3">
        <Field label="EPP">
          <select className="inp" value={f.recurso_id} onChange={(e) => setF({ ...f, recurso_id: +e.target.value })}>
            <option value={0}>Selecciona…</option>
            {(cat.data?.epp ?? []).map((r: Row) => <option key={r.id} value={r.id}>{r.nombre}</option>)}
          </select>
        </Field>
        <Field label="Motivo">
          <select className="inp" value={f.motivo} onChange={(e) => setF({ ...f, motivo: e.target.value })}>
            <option value="">Selecciona…</option>
            {MOTIVOS.map((m) => <option key={m}>{m}</option>)}
          </select>
        </Field>
        <Field label="Describe el problema"><textarea className="inp" rows={3} value={f.descripcion} onChange={(e) => setF({ ...f, descripcion: e.target.value })} /></Field>
        <PhotoField label="Foto del EPP" value={f.foto_id} onChange={(id) => setF({ ...f, foto_id: id })} />
        <Btn className="btn btn-primary w-full" disabled={!listo} onClick={submit}>Enviar solicitud</Btn>
        {!listo && <p className="text-center text-xs text-slate-500">Completa el EPP, el motivo, la descripción y la foto.</p>}
      </div>
    </Page>
  )
}

const LINEA = ['enviada', 'aprobada', 'entregada']

function Linea({ estado }: { estado: string }) {
  if (estado === 'rechazada' || estado === 'en_espera') return <Chip s={estado} />
  const i = LINEA.indexOf(estado)
  return (
    <div className="flex items-center gap-1 text-xs">
      {LINEA.map((s, j) => (
        <span key={s} className={`rounded px-1.5 py-0.5 ${j <= i ? 'bg-blue-700 text-white' : 'bg-slate-200 text-slate-500'}`}>{['Enviada', 'Aprobada', 'Entregada'][j]}</span>
      ))}
    </div>
  )
}

export function MisSolicitudes() {
  const q = useApi('/epp/mias')
  return (
    <Page title="Mis solicitudes" actions={<Link className="btn btn-primary" to="/solicitar">Nueva solicitud</Link>}>
      <Loading q={q} />
      {q.data?.length === 0 && <p className="muted">Aún no tienes solicitudes.</p>}
      <div className="space-y-2">
        {(q.data ?? []).map((e: Row) => (
          <Link key={e.id} to={`/epp/${e.id}`} className="card block hover:border-blue-300">
            <div className="flex items-center justify-between">
              <b>{e.recurso}</b>
              <span className="text-xs text-slate-500">{e.codigo} · {fmtD(e.created_at)}</span>
            </div>
            <div className="mt-2"><Linea estado={e.estado} /></div>
            {e.estado === 'en_espera' && <p className="mt-1 text-xs text-amber-700">Fecha estimada: {fmtD(e.fecha_espera)}</p>}
          </Link>
        ))}
      </div>
    </Page>
  )
}

export function EppDetalle({ me }: { me: Me }) {
  const { id } = useParams()
  const q = useApi<Row>(`/epp/${id}`)
  const act = useAction()
  const [modal, setModal] = useState<'' | 'espera' | 'rechazar' | 'entregar'>('')
  const [v, setV] = useState({ fecha: today(7), motivo: '', epp_anterior_devuelto: false, formato_firmado: false })
  const e = q.data
  const sst = me.rol === 'sst'
  const run = async (acc: string, msg: string) => {
    if (await act(`/epp/${id}/${acc}`, v, msg)) setModal('')
  }
  return (
    <Page back={sst ? '/epp' : '/mis-solicitudes'} title={e ? `${e.codigo} · ${e.recurso}` : 'Solicitud'} actions={e && <Chip s={e.estado} />}>
      <Loading q={q} />
      {e && (
        <div className="grid gap-4 md:grid-cols-2">
          <div className="card space-y-3">
            {e.estado === 'en_espera' && <Banner kind="warn">En espera. Fecha estimada de cambio: {fmtD(e.fecha_espera)}.</Banner>}
            {e.estado === 'rechazada' && <Banner kind="err">Rechazada: {e.motivo_rechazo}</Banner>}
            {e.estado === 'aprobada' && !sst && <Banner kind="ok">Aprobada. Acércate a SST para la entrega.</Banner>}
            {e.estado === 'entregada' && <Banner kind="ok">Entregada el {fmtDT(e.entregada_at)}.</Banner>}
            <dl className="grid grid-cols-[110px_1fr] gap-1 text-sm">
              {sst && <><dt className="text-slate-500">Trabajador</dt><dd>{e.trabajador}</dd><dt className="text-slate-500">Puesto</dt><dd>{e.puesto}</dd><dt className="text-slate-500">DNI</dt><dd>{e.dni}</dd></>}
              <dt className="text-slate-500">Motivo</dt><dd>{e.motivo}</dd>
              <dt className="text-slate-500">Descripción</dt><dd>{e.descripcion}</dd>
              <dt className="text-slate-500">Fecha</dt><dd>{fmtDT(e.created_at)}</dd>
              {sst && <><dt className="text-slate-500">Stock</dt><dd>{e.stock}</dd></>}
            </dl>
            <Img id={e.foto_id} className="h-48 w-full" />
            {sst && (
              <div className="flex flex-wrap gap-2">
                {(e.estado === 'enviada' || e.estado === 'en_espera') && <>
                  <Btn className="btn btn-primary" onClick={() => run('aprobar', 'Solicitud aprobada.')}>Aprobar</Btn>
                  {e.estado === 'enviada' && <Btn className="btn" onClick={() => setModal('espera')}>Poner en espera</Btn>}
                  <Btn className="btn btn-danger" onClick={() => setModal('rechazar')}>Rechazar</Btn>
                </>}
                {e.estado === 'aprobada' && <Btn className="btn btn-primary" onClick={() => setModal('entregar')}>Registrar entrega</Btn>}
              </div>
            )}
          </div>
          <div className="card"><h2 className="mb-2 font-semibold">Historial</h2><Historial items={e.historial} /></div>
        </div>
      )}
      {modal === 'espera' && (
        <Modal title="Poner en espera" onClose={() => setModal('')} footer={<Btn className="btn btn-primary" onClick={() => run('espera', 'Solicitud en espera.')}>Guardar</Btn>}>
          <Field label="Fecha estimada de cambio"><input type="date" className="inp" value={v.fecha} onChange={(x) => setV({ ...v, fecha: x.target.value })} /></Field>
        </Modal>
      )}
      {modal === 'rechazar' && (
        <Modal title="Rechazar solicitud" onClose={() => setModal('')} footer={<Btn className="btn btn-danger" onClick={() => run('rechazar', 'Solicitud rechazada.')}>Rechazar</Btn>}>
          <Field label="Motivo"><textarea className="inp" rows={3} value={v.motivo} onChange={(x) => setV({ ...v, motivo: x.target.value })} /></Field>
        </Modal>
      )}
      {modal === 'entregar' && (
        <Modal title="Registrar entrega" onClose={() => setModal('')} footer={<Btn className="btn btn-primary" onClick={() => run('entregar', 'Entrega registrada. El stock bajó en 1.')}>Confirmar entrega</Btn>}>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={v.epp_anterior_devuelto} onChange={(x) => setV({ ...v, epp_anterior_devuelto: x.target.checked })} /> El trabajador devolvió el EPP anterior</label>
          <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={v.formato_firmado} onChange={(x) => setV({ ...v, formato_firmado: x.target.checked })} /> Firmó el formato físico de entrega</label>
        </Modal>
      )}
    </Page>
  )
}

export function EppSst() {
  const [tab, setTab] = useState('enviada')
  const q = useApi(`/epp?estado=${tab}`)
  const nav = useNavigate()
  return (
    <Page title="Solicitudes de EPP">
      <Tabs value={tab} onChange={setTab} tabs={[['enviada', 'Nuevas'], ['aprobada', 'Aprobadas'], ['en_espera', 'En espera'], ['entregada', 'Entregadas'], ['rechazada', 'Rechazadas'], ['', 'Todas']]} />
      <Loading q={q} />
      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Código</th><th>Trabajador</th><th>EPP</th><th>Motivo</th><th>Fecha</th><th>Estado</th></tr></thead>
          <tbody>
            {(q.data ?? []).map((e: Row) => (
              <tr key={e.id} className="click" onClick={() => nav(`/epp/${e.id}`)}>
                <td>{e.codigo}</td><td>{e.trabajador}<div className="text-xs text-slate-500">{e.puesto}</div></td>
                <td>{e.recurso}</td><td>{e.motivo}</td><td>{fmtD(e.created_at)}</td><td><Chip s={e.estado} /></td>
              </tr>
            ))}
          </tbody>
        </table>
        {q.data?.length === 0 && <p className="muted p-4">No hay solicitudes en este estado.</p>}
      </div>
    </Page>
  )
}
