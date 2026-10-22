import { useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { api, type Me, type Row } from '../api/client'
import { Banner, Field, Modal, toast, useApi } from '../components/ui'

const DEMO: [string, string, string][] = [
  ['Trabajador', 'Luis Quispe', '71234567'],
  ['SST', 'Ana Torres', '45678912'],
  ['Mantenimiento', 'Carlos Mendoza', '43219876'],
  ['Administración', 'Rosa Chávez', '46781234'],
]

export function Login() {
  const qc = useQueryClient()
  const [dni, setDni] = useState('')
  const [pw, setPw] = useState('')
  const [err, setErr] = useState('')
  const [reg, setReg] = useState(false)

  const login = async (d = dni, p = pw) => {
    setErr('')
    if (!/^\d{8}$/.test(d)) return setErr('Ingresa un DNI de 8 dígitos.')
    try {
      const me = await api<Me>('/auth/login', { body: { dni: d, password: p } })
      qc.setQueryData(['/me'], me)
    } catch (e) {
      setErr((e as Error).message)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="card w-full max-w-sm">
        <h1 className="h1">SST Almacén</h1>
        <p className="muted mb-4">Ingresa con tu DNI.</p>
        <div className="mb-4 grid grid-cols-2 gap-2">
          {DEMO.map(([rol, nom, d]) => (
            <button key={d} className="btn flex-col items-start text-left" onClick={() => { setDni(d); setPw('demo1234'); login(d, 'demo1234') }}>
              <span className="text-xs text-slate-500">{rol}</span>
              <span>{nom}</span>
            </button>
          ))}
        </div>
        <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); login() }}>
          <Field label="DNI"><input className="inp" inputMode="numeric" maxLength={8} value={dni} onChange={(e) => setDni(e.target.value)} placeholder="8 dígitos" /></Field>
          <Field label="Contraseña"><input className="inp" type="password" value={pw} onChange={(e) => setPw(e.target.value)} /></Field>
          {err && <Banner kind="err">{err}</Banner>}
          <button className="btn btn-primary w-full">Ingresar</button>
        </form>
        <p className="muted mt-4 text-center">¿Primera vez? <button className="text-blue-700 underline" onClick={() => setReg(true)}>Regístrate con tu DNI</button></p>
        <p className="mt-2 text-center text-xs text-slate-400">Usuarios de prueba: contraseña demo1234</p>
      </div>
      {reg && <Registro onClose={() => setReg(false)} />}
    </div>
  )
}

function Registro({ onClose }: { onClose: () => void }) {
  const areas = useApi('/areas')
  const [f, setF] = useState({ dni: '', nombres: '', correo: '', celular: '', area_id: 0, password: '' })
  const [err, setErr] = useState('')
  const set = (k: string, v: string | number) => setF({ ...f, [k]: v })
  const submit = async () => {
    setErr('')
    if (!/^\d{8}$/.test(f.dni)) return setErr('El DNI debe tener 8 dígitos.')
    try {
      await api('/auth/register', { body: f })
      toast('Registro enviado. SST te asignará a tu puesto.')
      onClose()
    } catch (e) {
      setErr((e as Error).message)
    }
  }
  return (
    <Modal title="Registro con DNI" onClose={onClose} footer={<><button className="btn" onClick={onClose}>Cancelar</button><button className="btn btn-primary" onClick={submit}>Registrarme</button></>}>
      <p className="muted">Todo el personal, operativo y administrativo, se registra con su DNI. SST te asignará al puesto que ocupas.</p>
      <div className="grid grid-cols-2 gap-3">
        <Field label="DNI"><input className="inp" maxLength={8} inputMode="numeric" value={f.dni} onChange={(e) => set('dni', e.target.value)} /></Field>
        <Field label="Celular"><input className="inp" value={f.celular} onChange={(e) => set('celular', e.target.value)} /></Field>
      </div>
      <Field label="Nombres y apellidos"><input className="inp" value={f.nombres} onChange={(e) => set('nombres', e.target.value)} /></Field>
      <Field label="Correo"><input className="inp" type="email" value={f.correo} onChange={(e) => set('correo', e.target.value)} /></Field>
      <Field label="Área">
        <select className="inp" value={f.area_id} onChange={(e) => set('area_id', +e.target.value)}>
          <option value={0}>Selecciona…</option>
          {(areas.data ?? []).map((a: Row) => <option key={a.id} value={a.id}>{a.nombre}</option>)}
        </select>
      </Field>
      <Field label="Contraseña"><input className="inp" type="password" value={f.password} onChange={(e) => set('password', e.target.value)} /></Field>
      {err && <Banner kind="err">{err}</Banner>}
    </Modal>
  )
}
