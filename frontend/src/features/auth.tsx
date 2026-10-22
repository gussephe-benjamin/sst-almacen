import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, type Me, type Row } from '../api/client'
import { Btn, Banner, Field, Modal, useApi } from '../components/ui'

export function Login() {
  const qc = useQueryClient()
  const [dni, setDni] = useState('')
  const [pw, setPw] = useState('')
  const [err, setErr] = useState('')
  const [reg, setReg] = useState(false)
  const [busy, setBusy] = useState(false)
  useEffect(() => { document.title = 'Ingresar · SST Almacén' }, [])

  const login = async () => {
    setErr('')
    if (!/^\d{8}$/.test(dni)) return setErr('Ingresa un DNI de 8 dígitos.')
    if (!pw) return setErr('Ingresa tu contraseña.')
    setBusy(true)
    try {
      const me = await api<Me>('/auth/login', { body: { dni, password: pw } })
      qc.setQueryData(['/me'], me)
    } catch (e) {
      setErr((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="card w-full max-w-sm">
        <h1 className="h1">SST Almacén</h1>
        <p className="muted mb-4">Ingresa con tu DNI.</p>
        <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); return login() }}>
          <Field label="DNI"><input className="inp" inputMode="numeric" autoComplete="username" autoFocus maxLength={8} value={dni} onChange={(e) => setDni(e.target.value.replace(/\D/g, ''))} placeholder="8 dígitos" /></Field>
          <Field label="Contraseña"><input className="inp" type="password" autoComplete="current-password" value={pw} onChange={(e) => setPw(e.target.value)} /></Field>
          {err && <Banner kind="err">{err}</Banner>}
          <button className="btn btn-primary w-full" disabled={busy}>{busy ? 'Ingresando…' : 'Ingresar'}</button>
        </form>
        <p className="muted mt-4 text-center">¿Primera vez? <Btn className="text-blue-700 underline" onClick={() => setReg(true)}>Regístrate con tu DNI</Btn></p>
        <p className="mt-2 text-center text-xs text-slate-500">¿Olvidaste tu contraseña? Pide al administrador que la restablezca.</p>
      </div>
      {reg && <Registro onClose={() => setReg(false)} />}
    </div>
  )
}

function Registro({ onClose }: { onClose: () => void }) {
  const areas = useApi('/areas')
  const [f, setF] = useState({ dni: '', nombres: '', correo: '', celular: '', area_id: 0, password: '' })
  const [err, setErr] = useState('')
  const [listo, setListo] = useState(false)
  const set = (k: string, v: string | number) => { setErr(''); setF({ ...f, [k]: v }) }
  const submit = async () => {
    if (!/^\d{8}$/.test(f.dni)) return setErr('El DNI debe tener 8 dígitos.')
    if (!f.nombres.trim() || !f.correo.trim() || !f.celular.trim()) return setErr('Completa nombres, correo y celular.')
    if (!f.area_id) return setErr('Elige el área donde trabajas.')
    if (f.password.length < 8) return setErr('La contraseña debe tener al menos 8 caracteres.')
    try {
      await api('/auth/register', { body: f })
      setListo(true)
    } catch (e) {
      setErr((e as Error).message)
    }
  }
  if (listo) {
    return (
      <Modal title="Registro enviado" onClose={onClose} footer={<button className="btn btn-primary" onClick={onClose}>Entendido</button>}>
        <Banner kind="ok">Tu registro quedó pendiente. Podrás ingresar con tu DNI y contraseña cuando te asignen a tu puesto.</Banner>
      </Modal>
    )
  }
  return (
    <Modal title="Registro con DNI" onClose={onClose} footer={<><button className="btn" onClick={onClose}>Cancelar</button><Btn className="btn btn-primary" onClick={submit}>Registrarme</Btn></>}>
      <p className="muted">Todo el personal, operativo y administrativo, se registra con su DNI. Luego te asignarán al puesto que ocupas.</p>
      <div className="grid grid-cols-2 gap-3">
        <Field label="DNI"><input className="inp" maxLength={8} inputMode="numeric" value={f.dni} onChange={(e) => set('dni', e.target.value.replace(/\D/g, ''))} /></Field>
        <Field label="Celular"><input className="inp" inputMode="tel" value={f.celular} onChange={(e) => set('celular', e.target.value)} /></Field>
      </div>
      <Field label="Nombres y apellidos"><input className="inp" value={f.nombres} onChange={(e) => set('nombres', e.target.value)} /></Field>
      <Field label="Correo"><input className="inp" type="email" value={f.correo} onChange={(e) => set('correo', e.target.value)} /></Field>
      <Field label="Área">
        <select className="inp" value={f.area_id} onChange={(e) => set('area_id', +e.target.value)}>
          <option value={0}>Selecciona…</option>
          {(areas.data ?? []).map((a: Row) => <option key={a.id} value={a.id}>{a.nombre}</option>)}
        </select>
      </Field>
      <Field label="Contraseña (mínimo 8 caracteres)"><input className="inp" type="password" autoComplete="new-password" value={f.password} onChange={(e) => set('password', e.target.value)} /></Field>
      {err && <Banner kind="err">{err}</Banner>}
    </Modal>
  )
}
