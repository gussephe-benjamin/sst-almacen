import { useMemo, useState } from 'react'
import { Copy, KeyRound, Plus, RefreshCw, Search } from 'lucide-react'
import type { Me, Row } from '../api/client'
import { Banner, Btn, Chip, Field, Loading, Modal, Page, Tabs, confirmar, fmtD, fmtDT, toast, useAction, useApi } from '../components/ui'

const ROLES: [string, string][] = [['trabajador', 'Trabajador'], ['sst', 'SST'], ['mantenimiento', 'Mantenimiento'], ['administracion', 'Administración']]
const ROL_LABEL: Record<string, string> = { ...Object.fromEntries(ROLES), admin: 'Administrador' }
const ROL_CHIP: Record<string, string> = { trabajador: 'enviada', sst: 'aprobada', mantenimiento: 'en_espera', administracion: 'por_validar', admin: 'pendiente' }

const genPassword = () => {
  const c = 'abcdefghjkmnpqrstuvwxyz23456789'
  return Array.from(crypto.getRandomValues(new Uint32Array(8)), (n) => c[n % c.length]).join('')
}

function PasswordField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <Field label="Contraseña inicial (mínimo 6 caracteres)">
      <div className="flex gap-2">
        <input className="inp font-mono" value={value} onChange={(e) => onChange(e.target.value)} />
        <button type="button" className="btn" title="Generar contraseña" onClick={() => onChange(genPassword())}><RefreshCw size={14} /> Generar</button>
      </div>
    </Field>
  )
}

/** Muestra las credenciales recién creadas para entregarlas a la persona. */
function Credenciales({ c, onClose }: { c: { nombre: string; dni: string; password: string }; onClose: () => void }) {
  const texto = `Usuario (DNI): ${c.dni}\nContraseña: ${c.password}`
  return (
    <Modal title="Credenciales listas" onClose={onClose} footer={<button className="btn btn-primary" onClick={onClose}>Listo</button>}>
      <p className="text-sm">Entrega estos datos a <b>{c.nombre}</b>. La contraseña no se volverá a mostrar.</p>
      <pre className="rounded-md bg-slate-100 p-3 text-sm">{texto}</pre>
      <button className="btn" onClick={() => navigator.clipboard.writeText(texto).then(() => toast('Copiado.'))}><Copy size={14} /> Copiar</button>
    </Modal>
  )
}

export function Cuentas({ me }: { me: Me }) {
  const q = useApi<Row>('/admin/cuentas')
  const areas = useApi('/areas')
  const act = useAction()
  const [rol, setRol] = useState('')
  const [busca, setBusca] = useState('')
  const [soloVacantes, setSoloVacantes] = useState(false)
  const [crear, setCrear] = useState<{ puesto_id: number; rol: string } | null>(null)
  const [asignar, setAsignar] = useState<Row | null>(null)
  const [reset, setReset] = useState<Row | null>(null)
  const [hist, setHist] = useState<Row | null>(null)
  const [cred, setCred] = useState<{ nombre: string; dni: string; password: string } | null>(null)

  const puestos: Row[] = q.data?.puestos ?? []
  const sinPuesto: Row[] = q.data?.sin_puesto ?? []
  const filtrados = useMemo(() => {
    const t = busca.trim().toLowerCase()
    return puestos.filter((p) =>
      (!rol || p.rol === rol) && (!soloVacantes || !p.persona) &&
      (!t || [p.nombre, p.persona, p.dni, p.area].some((x) => String(x ?? '').toLowerCase().includes(t))))
  }, [puestos, rol, busca, soloVacantes])
  const count = (r: string) => puestos.filter((p) => p.rol === r).length

  return (
    <Page title="Cuentas y puestos"
      sub="Las cuentas pertenecen al puesto. Al cambiar al personal, la persona nueva hereda las solicitudes, tareas y avisos del puesto. Las cuentas de administrador se definen en el archivo .env."
      actions={<button className="btn btn-primary" onClick={() => setCrear({ puesto_id: 0, rol: rol && rol !== 'admin' ? rol : 'trabajador' })}><Plus size={14} /> Crear cuenta</button>}>
      <Loading q={q} />

      {sinPuesto.length > 0 && (
        <div className="card mb-4">
          <h2 className="mb-2 font-semibold">Personas sin puesto <Chip s="en_espera">{sinPuesto.length}</Chip></h2>
          <div className="overflow-x-auto">
            <table className="tbl"><tbody>
              {sinPuesto.map((u) => (
                <tr key={u.id}>
                  <td><b>{u.nombres}</b><div className="text-xs text-slate-500">DNI {u.dni}</div></td>
                  <td>{u.estado === 'pendiente' ? <Chip s="en_espera">Registro nuevo</Chip> : <Chip s="pendiente">Sin puesto actual</Chip>}</td>
                  <td className="text-xs">{u.correo || '—'}<div className="text-slate-500">{u.celular}</div></td>
                  <td className="text-xs">{u.area ?? '—'}<div className="text-slate-500">{fmtDT(u.created_at)}</div></td>
                  <td className="text-right"><button className="btn btn-sm btn-primary" onClick={() => setAsignar(u)}>Asignar a un puesto</button></td>
                </tr>
              ))}
            </tbody></table>
          </div>
        </div>
      )}

      <Tabs value={rol} onChange={setRol} tabs={[['', `Todos (${puestos.length})`], ...ROLES.map(([v, l]): [string, string] => [v, `${l} (${count(v)})`]), ['admin', `Administrador (${count('admin')})`]]} />
      <div className="mb-3 flex flex-wrap items-center gap-3">
        <div className="relative min-w-60 flex-1">
          <Search size={14} className="absolute left-3 top-3 text-slate-400" />
          <input className="inp pl-8" placeholder="Buscar por puesto, persona, DNI o área" value={busca} onChange={(e) => setBusca(e.target.value)} />
        </div>
        <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={soloVacantes} onChange={(e) => setSoloVacantes(e.target.checked)} /> Solo vacantes</label>
      </div>

      <div className="card overflow-x-auto p-0">
        <table className="tbl">
          <thead><tr><th>Puesto</th><th>Rol</th><th>Persona actual</th><th>Contacto</th><th>Desde</th><th></th></tr></thead>
          <tbody>
            {filtrados.map((p) => {
              const esAdmin = p.rol === 'admin'
              const propio = p.id === me.puesto_id
              return (
                <tr key={p.id}>
                  <td><b>{p.nombre}</b><div className="text-xs text-slate-500">{p.area}</div></td>
                  <td><Chip s={ROL_CHIP[p.rol]}>{ROL_LABEL[p.rol]}</Chip></td>
                  <td>{p.persona ?? <span className="text-amber-700">Vacante</span>}{p.dni && <div className="text-xs text-slate-500">DNI {p.dni}</div>}</td>
                  <td className="text-xs">{p.celular || '—'}<div className="text-slate-500">{p.correo}</div></td>
                  <td className="text-xs">{fmtD(p.desde)}</td>
                  <td className="whitespace-nowrap text-right">
                    {esAdmin && !propio ? <span className="text-xs text-slate-400">Se gestiona en .env</span> : <div className="inline-flex flex-wrap justify-end gap-1">
                      {!esAdmin && <button className="btn btn-sm" onClick={() => setCrear({ puesto_id: p.id, rol: p.rol })}>{p.persona ? 'Cambiar personal' : 'Asignar persona'}</button>}
                      {p.persona && <button className="btn btn-sm" title="Restablecer contraseña" onClick={() => setReset(p)}><KeyRound size={12} /> Contraseña</button>}
                      {p.persona && !esAdmin && (
                        <Btn className="btn btn-sm btn-danger" onClick={() => confirmar(`¿Retirar a ${p.persona} del puesto "${p.nombre}"? Ya no podrá ingresar y el puesto quedará vacante.`)
                          ? act(`/admin/puestos/${p.id}/liberar`, {}, 'Puesto liberado.') : undefined}>Liberar</Btn>
                      )}
                      <button className="btn btn-sm" onClick={() => setHist(p)}>Historial{p.anteriores > 0 && ` (${p.anteriores})`}</button>
                    </div>}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
        {!q.isLoading && filtrados.length === 0 && <p className="muted p-4">No hay puestos que coincidan con la búsqueda.</p>}
      </div>

      {crear && <CrearCuenta init={crear} puestos={puestos} areas={areas.data ?? []} onClose={() => setCrear(null)} onDone={(c) => { setCrear(null); setCred(c) }} />}
      {asignar && <AsignarPuesto u={asignar} puestos={puestos} onClose={() => setAsignar(null)} />}
      {reset && (
        <ResetPassword p={reset} onClose={() => setReset(null)} onDone={(pw) => { setCred({ nombre: reset.persona, dni: reset.dni, password: pw }); setReset(null) }} />
      )}
      {hist && <Historial p={hist} onClose={() => setHist(null)} />}
      {cred && <Credenciales c={cred} onClose={() => setCred(null)} />}
    </Page>
  )
}

function CrearCuenta({ init, puestos, areas, onClose, onDone }: {
  init: { puesto_id: number; rol: string }; puestos: Row[]; areas: Row[]
  onClose: () => void; onDone: (c: { nombre: string; dni: string; password: string }) => void
}) {
  const act = useAction()
  const [f, setF] = useState({ rol: init.rol, puesto_id: init.puesto_id, puesto_nombre: '', area_id: 0, dni: '', nombres: '', correo: '', celular: '', password: genPassword() })
  const [err, setErr] = useState('')
  const set = (k: string, v: string | number) => { setErr(''); setF({ ...f, [k]: v }) }
  const opciones = puestos.filter((p) => p.rol === f.rol).sort((a, b) => Number(!!a.persona) - Number(!!b.persona))
  const elegido = puestos.find((p) => p.id === f.puesto_id)

  const save = async () => {
    if (!/^\d{8}$/.test(f.dni)) return setErr('El DNI debe tener 8 dígitos.')
    if (!f.nombres.trim()) return setErr('Ingresa los nombres y apellidos.')
    if (f.password.length < 6) return setErr('La contraseña debe tener al menos 6 caracteres.')
    if (!f.puesto_id && (!f.puesto_nombre.trim() || !f.area_id)) return setErr('Indica el nombre del puesto nuevo y su área.')
    if (elegido?.persona && !confirmar(`${elegido.persona} saldrá del puesto "${elegido.nombre}" y ya no podrá ingresar. ¿Continuar?`)) return
    if (await act('/admin/cuentas', f, 'Cuenta creada.')) onDone({ nombre: f.nombres, dni: f.dni, password: f.password })
  }

  return (
    <Modal title={init.puesto_id ? `Nueva persona para ${elegido?.nombre}` : 'Crear cuenta'} onClose={onClose}
      footer={<><button className="btn" onClick={onClose}>Cancelar</button><Btn className="btn btn-primary" onClick={save}>Crear cuenta</Btn></>}>
      {!init.puesto_id && <>
        <Field label="Rol">
          <select className="inp" value={f.rol} onChange={(e) => setF({ ...f, rol: e.target.value, puesto_id: 0 })}>
            {ROLES.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
          </select>
        </Field>
        <Field label="Puesto">
          <select className="inp" value={f.puesto_id} onChange={(e) => set('puesto_id', +e.target.value)}>
            <option value={0}>+ Crear un puesto nuevo</option>
            {opciones.map((p) => <option key={p.id} value={p.id}>{p.nombre} {p.persona ? `(hoy: ${p.persona})` : '(vacante)'}</option>)}
          </select>
        </Field>
        {!f.puesto_id && (
          <div className="grid grid-cols-2 gap-3">
            <Field label="Nombre del puesto"><input className="inp" placeholder="Ej. Operario de almacén 3" value={f.puesto_nombre} onChange={(e) => set('puesto_nombre', e.target.value)} /></Field>
            <Field label="Área">
              <select className="inp" value={f.area_id} onChange={(e) => set('area_id', +e.target.value)}>
                <option value={0}>Selecciona…</option>
                {areas.map((a) => <option key={a.id} value={a.id}>{a.nombre}</option>)}
              </select>
            </Field>
          </div>
        )}
      </>}
      {elegido?.persona && <Banner kind="warn"><b>{elegido.persona}</b> ocupa este puesto. Al crear la cuenta saldrá del puesto y la persona nueva heredará su historial.</Banner>}
      <div className="grid grid-cols-2 gap-3">
        <Field label="DNI"><input className="inp" inputMode="numeric" maxLength={8} value={f.dni} onChange={(e) => set('dni', e.target.value.replace(/\D/g, ''))} /></Field>
        <Field label="Celular"><input className="inp" inputMode="tel" value={f.celular} onChange={(e) => set('celular', e.target.value)} /></Field>
      </div>
      <Field label="Nombres y apellidos"><input className="inp" value={f.nombres} onChange={(e) => set('nombres', e.target.value)} /></Field>
      <Field label="Correo"><input className="inp" type="email" value={f.correo} onChange={(e) => set('correo', e.target.value)} /></Field>
      <PasswordField value={f.password} onChange={(v) => set('password', v)} />
      {err && <Banner kind="err">{err}</Banner>}
    </Modal>
  )
}

function AsignarPuesto({ u, puestos, onClose }: { u: Row; puestos: Row[]; onClose: () => void }) {
  const act = useAction()
  const opciones = puestos.filter((p) => p.rol !== 'admin').sort((a, b) => Number(!!a.persona) - Number(!!b.persona))
  const [pid, setPid] = useState(0)
  const elegido = puestos.find((p) => p.id === pid)
  const save = async () => {
    if (!pid) return toast('Elige un puesto.', true)
    if (await act(`/admin/puestos/${pid}/asignar`, { usuario_id: u.id }, `${u.nombres} fue asignado al puesto.`)) onClose()
  }
  return (
    <Modal title={`Asignar a ${u.nombres}`} onClose={onClose}
      footer={<><button className="btn" onClick={onClose}>Cancelar</button><Btn className="btn btn-primary" onClick={save}>Asignar</Btn></>}>
      <p className="muted">DNI {u.dni}{u.area && ` · se registró en ${u.area}`}</p>
      <Field label="Puesto">
        <select className="inp" value={pid} onChange={(e) => setPid(+e.target.value)}>
          <option value={0}>Selecciona…</option>
          {ROLES.map(([r, l]) => (
            <optgroup key={r} label={l}>
              {opciones.filter((p) => p.rol === r).map((p) => <option key={p.id} value={p.id}>{p.nombre} {p.persona ? `(hoy: ${p.persona})` : '(vacante)'}</option>)}
            </optgroup>
          ))}
        </select>
      </Field>
      {elegido?.persona && <Banner kind="warn"><b>{elegido.persona}</b> saldrá del puesto y ya no podrá ingresar.</Banner>}
      <p className="muted">La persona ingresará con su DNI y la contraseña que eligió al registrarse.</p>
    </Modal>
  )
}

function ResetPassword({ p, onClose, onDone }: { p: Row; onClose: () => void; onDone: (pw: string) => void }) {
  const act = useAction()
  const [pw, setPw] = useState(genPassword())
  return (
    <Modal title={`Contraseña de ${p.persona}`} onClose={onClose}
      footer={<><button className="btn" onClick={onClose}>Cancelar</button>
        <Btn className="btn btn-primary" onClick={async () => { if (await act(`/admin/usuarios/${p.usuario_id}/password`, { password: pw }, 'Contraseña restablecida.')) onDone(pw) }}>Restablecer</Btn></>}>
      <PasswordField value={pw} onChange={setPw} />
    </Modal>
  )
}

function Historial({ p, onClose }: { p: Row; onClose: () => void }) {
  const h = useApi<Row>(`/admin/puestos/${p.id}/historial`)
  return (
    <Modal title={`Historial: ${p.nombre}`} onClose={onClose}>
      <Loading q={h} />
      <h3 className="font-semibold">Personas en el puesto</h3>
      <ul className="text-sm">
        {(h.data?.asignaciones ?? []).map((a: Row, i: number) => <li key={i}>{a.persona} (DNI {a.dni}): {fmtD(a.desde)} – {a.hasta ? fmtD(a.hasta) : 'actual'}</li>)}
        {h.data?.asignaciones?.length === 0 && <li className="muted">Nadie ha ocupado este puesto.</li>}
      </ul>
      <h3 className="font-semibold">Acciones del puesto</h3>
      <ul className="max-h-64 space-y-1 overflow-auto text-sm">
        {(h.data?.acciones ?? []).map((a: Row, i: number) => <li key={i}>{a.accion} <span className="text-xs text-slate-500">· {a.quien} · {fmtDT(a.fecha)}</span></li>)}
      </ul>
    </Modal>
  )
}
