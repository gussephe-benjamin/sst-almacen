import { useEffect, useState, useSyncExternalStore, type ButtonHTMLAttributes, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Camera, X } from 'lucide-react'
import { api, fileUrl, upload, type Row } from '../api/client'

// ---------- Datos ----------

export function useApi<T = Row[]>(path: string | null, refetchMs?: number) {
  return useQuery<T>({ queryKey: [path], queryFn: () => api<T>(path!), enabled: !!path, refetchInterval: refetchMs })
}

/** Pide confirmación antes de una acción que no se puede deshacer. */
export const confirmar = (msg: string) => window.confirm(msg)

/** Botón que se deshabilita mientras su acción está en curso (evita dobles envíos). */
export function Btn({ onClick, disabled, children, ...p }: ButtonHTMLAttributes<HTMLButtonElement>) {
  const [busy, setBusy] = useState(false)
  return (
    <button {...p} disabled={disabled || busy} aria-busy={busy}
      onClick={async (e) => {
        const r = onClick?.(e) as unknown
        if (r instanceof Promise) {
          setBusy(true)
          try { await r } finally { setBusy(false) }
        }
      }}>
      {children}{busy && '…'}
    </button>
  )
}

/** Ejecuta una acción POST/PUT/DELETE, refresca los datos y muestra el resultado. */
export function useAction() {
  const qc = useQueryClient()
  return async (path: string, body: unknown = {}, okMsg?: string, method?: string) => {
    try {
      const r = await api(path, { body, method })
      await qc.invalidateQueries()
      if (okMsg) toast(okMsg)
      return r
    } catch (e) {
      toast((e as Error).message, true)
      return null
    }
  }
}

// ---------- Toast ----------

let toastState: { msg: string; err: boolean } | null = null
const subs = new Set<() => void>()
export function toast(msg: string, err = false) {
  toastState = { msg, err }
  subs.forEach((f) => f())
  setTimeout(() => {
    toastState = null
    subs.forEach((f) => f())
  }, 3500)
}
export function Toast() {
  const t = useSyncExternalStore((f) => (subs.add(f), () => subs.delete(f)), () => toastState)
  if (!t) return null
  return (
    <div className={`fixed bottom-4 left-1/2 z-50 -translate-x-1/2 rounded-md px-4 py-2 text-sm text-white shadow-lg ${t.err ? 'bg-red-600' : 'bg-slate-800'}`}>
      {t.msg}
    </div>
  )
}

// ---------- Formato ----------

export const fmtD = (s?: string | null) => (s ? new Date(s.length === 10 ? s + 'T12:00:00' : s).toLocaleDateString('es-PE') : '—')
export const fmtDT = (s?: string | null) => (s ? new Date(s).toLocaleString('es-PE', { dateStyle: 'short', timeStyle: 'short' }) : '—')
// Fecha local (no UTC): de noche en Perú, la fecha UTC ya es la del día siguiente.
export const today = (plusDays = 0) => new Date(Date.now() + plusDays * 864e5 - new Date().getTimezoneOffset() * 6e4).toISOString().slice(0, 10)

const LABELS: Record<string, [string, string]> = {
  enviada: ['Enviada', 'bg-blue-100 text-blue-800'],
  aprobada: ['Aprobada', 'bg-emerald-100 text-emerald-800'],
  en_espera: ['En espera', 'bg-amber-100 text-amber-800'],
  rechazada: ['Rechazada', 'bg-red-100 text-red-800'],
  entregada: ['Entregada', 'bg-slate-200 text-slate-700'],
  registrado: ['Registrado', 'bg-blue-100 text-blue-800'],
  en_atencion: ['En atención', 'bg-amber-100 text-amber-800'],
  cerrado: ['Cerrado', 'bg-emerald-100 text-emerald-800'],
  descartado: ['Descartado', 'bg-slate-200 text-slate-600'],
  asignada: ['Asignada', 'bg-blue-100 text-blue-800'],
  modificacion: ['Modificación solicitada', 'bg-purple-100 text-purple-800'],
  en_proceso: ['En proceso', 'bg-amber-100 text-amber-800'],
  por_validar: ['Por validar', 'bg-cyan-100 text-cyan-800'],
  devuelta: ['Devuelta', 'bg-red-100 text-red-800'],
  cerrada: ['Cerrada', 'bg-emerald-100 text-emerald-800'],
  pendiente: ['Pendiente', 'bg-slate-200 text-slate-700'],
  en_curso: ['En curso', 'bg-amber-100 text-amber-800'],
  completada: ['Completada', 'bg-emerald-100 text-emerald-800'],
  alto: ['Alto', 'bg-red-100 text-red-800'],
  medio: ['Medio', 'bg-amber-100 text-amber-800'],
  bajo: ['Bajo', 'bg-emerald-100 text-emerald-800'],
  alta: ['Alta', 'bg-red-50 text-red-700'],
  media: ['Media', 'bg-amber-50 text-amber-700'],
  baja: ['Baja', 'bg-emerald-50 text-emerald-700'],
}
export const label = (s: string) => LABELS[s]?.[0] ?? s

/** "1 recurso", "3 recursos". */
export const plural = (n: number, uno: string, varios: string) => `${n} ${n === 1 ? uno : varios}`

export function Chip({ s, children }: { s?: string; children?: ReactNode }) {
  const cls = (s && LABELS[s]?.[1]) || 'bg-slate-100 text-slate-700'
  return <span className={`inline-block whitespace-nowrap rounded-full px-2 py-0.5 text-xs font-semibold ${cls}`}>{children ?? label(s ?? '')}</span>
}

// ---------- Layout ----------

export function Page({ title, sub, actions, back, children }: { title: string; sub?: ReactNode; actions?: ReactNode; back?: string; children: ReactNode }) {
  useEffect(() => { document.title = `${title} · SST Almacén` }, [title])
  return (
    <div className="mx-auto max-w-6xl p-4 md:p-6">
      {back && <Link to={back} className="mb-2 inline-flex items-center gap-1 text-sm text-blue-700 hover:underline"><ArrowLeft size={14} /> Volver</Link>}
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="h1">{title}</h1>
          {sub && <p className="muted mt-1">{sub}</p>}
        </div>
        {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
      </div>
      {children}
    </div>
  )
}

export function Tabs({ tabs, value, onChange }: { tabs: [string, string][]; value: string; onChange: (v: string) => void }) {
  return (
    <div className="mb-3 flex flex-wrap gap-1 border-b border-slate-200">
      {tabs.map(([v, l]) => (
        <button key={v} onClick={() => onChange(v)}
          className={`-mb-px border-b-2 px-3 py-2 text-sm ${value === v ? 'border-blue-700 font-semibold text-blue-700' : 'border-transparent text-slate-500'}`}>
          {l}
        </button>
      ))}
    </div>
  )
}

export function Modal({ title, onClose, children, footer }: { title: string; onClose: () => void; children: ReactNode; footer?: ReactNode }) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose])
  return (
    <div className="fixed inset-0 z-40 flex items-end justify-center bg-black/40 sm:items-center" onClick={onClose}>
      <div className="max-h-[92vh] w-full max-w-lg overflow-auto rounded-t-xl bg-white p-5 sm:rounded-xl" onClick={(e) => e.stopPropagation()}>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-lg font-bold">{title}</h2>
          <button onClick={onClose} className="text-slate-400 hover:text-slate-700" aria-label="Cerrar"><X size={18} /></button>
        </div>
        <div className="space-y-3">{children}</div>
        {footer && <div className="mt-4 flex justify-end gap-2">{footer}</div>}
      </div>
    </div>
  )
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="lbl">{label}</span>
      {children}
    </label>
  )
}

export function Banner({ kind = 'info', children }: { kind?: 'info' | 'warn' | 'err' | 'ok'; children: ReactNode }) {
  const cls = { info: 'bg-blue-50 border-blue-200', warn: 'bg-amber-50 border-amber-200', err: 'bg-red-50 border-red-200', ok: 'bg-emerald-50 border-emerald-200' }[kind]
  return <div className={`mb-3 rounded-md border p-3 text-sm ${cls}`}>{children}</div>
}

export function Img({ id, className = '' }: { id?: string | null; className?: string }) {
  if (!id) return null
  return (
    <a href={fileUrl(id)} target="_blank" rel="noreferrer">
      <img src={fileUrl(id)} className={`rounded-md border border-slate-200 object-cover ${className}`} />
    </a>
  )
}

/** Campo de foto: en el celular abre la cámara. Sube la imagen y devuelve su id. */
export function PhotoField({ value, onChange, label = 'Foto' }: { value: string; onChange: (id: string) => void; label?: string }) {
  const [busy, setBusy] = useState(false)
  return (
    <div>
      <span className="lbl">{label}</span>
      {value ? (
        <div className="flex items-center gap-2">
          <Img id={value} className="h-24 w-32" />
          <button type="button" className="btn btn-sm" onClick={() => onChange('')}>Cambiar</button>
        </div>
      ) : (
        <label className="btn w-full cursor-pointer border-dashed py-6">
          <Camera size={16} /> {busy ? 'Subiendo…' : 'Tomar o elegir foto'}
          <input type="file" accept="image/*" capture="environment" className="hidden"
            onChange={async (e) => {
              const f = e.target.files?.[0]
              if (!f) return
              if (f.size > 5 * 1024 * 1024) return toast('La imagen no debe superar 5 MB.', true)
              setBusy(true)
              try { onChange(await upload(f)) } catch (err) { toast((err as Error).message, true) } finally { setBusy(false) }
            }} />
        </label>
      )}
    </div>
  )
}

export function Historial({ items }: { items?: Row[] }) {
  if (!items?.length) return <p className="muted">Sin movimientos.</p>
  return (
    <ol className="space-y-2 border-l-2 border-slate-200 pl-4">
      {items.map((h, i) => (
        <li key={i} className="text-sm">
          <div>{h.accion}</div>
          <div className="text-xs text-slate-500">{fmtDT(h.fecha)} · {h.quien} ({h.puesto})</div>
        </li>
      ))}
    </ol>
  )
}

export function Loading({ q }: { q: { isLoading: boolean; error: unknown } }) {
  if (q.isLoading) return <p className="muted p-4">Cargando…</p>
  if (q.error) return <Banner kind="err">{(q.error as Error).message}</Banner>
  return null
}
