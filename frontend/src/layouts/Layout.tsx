import { useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Bell, BookOpen, Box, ClipboardList, HardHat, LayoutDashboard, ListChecks, LogOut, Menu, Search, UserCog, Users } from 'lucide-react'
import { api, type Me, type Row } from '../api/client'
import { Toast, useApi } from '../components/ui'

type Item = [string, string, typeof Bell, number?]

export function Layout({ me }: { me: Me }) {
  const qc = useQueryClient()
  const nav = useNavigate()
  const [open, setOpen] = useState(false)
  const notifs = useApi('/notificaciones', 30_000)
  const unread = (notifs.data ?? []).filter((n: Row) => !n.leida).length
  const dash = useApi<Row>(me.rol === 'sst' ? '/dashboard' : null, 60_000)
  const cuentas = useApi<Row>(me.rol === 'admin' ? '/admin/cuentas' : null)
  const d = dash.data

  const menus: Record<Me['rol'], Item[]> = {
    trabajador: [['/solicitar', 'Solicitar EPP', HardHat], ['/mis-solicitudes', 'Mis solicitudes', ClipboardList], ['/biblioteca', 'Biblioteca', BookOpen]],
    sst: [
      ['/dashboard', 'Dashboard', LayoutDashboard], ['/hallazgos', 'Hallazgos', Search, d?.abiertos],
      ['/tareas', 'Tareas a áreas', ListChecks, (d?.tareas_por_validar ?? 0) + (d?.modificaciones ?? 0)],
      ['/epp', 'Solicitudes de EPP', HardHat, d?.epp_nuevas], ['/inventario', 'Inventario', Box, d?.bajo_minimo?.length],
      ['/internas', 'Tareas internas', ClipboardList], ['/biblioteca', 'Biblioteca', BookOpen], ['/puestos', 'Puestos y usuarios', Users],
    ],
    mantenimiento: [['/tareas', 'Mis tareas', ListChecks], ['/biblioteca', 'Biblioteca', BookOpen]],
    administracion: [['/tareas', 'Mis tareas', ListChecks], ['/biblioteca', 'Biblioteca', BookOpen]],
    admin: [['/cuentas', 'Cuentas y puestos', UserCog, (cuentas.data?.sin_puesto ?? []).filter((u: Row) => u.estado === 'pendiente').length]],
  }
  const items: Item[] = [...menus[me.rol], ['/notificaciones', 'Notificaciones', Bell, unread]]
  const rolLabel = { trabajador: 'Trabajador', sst: 'SST', mantenimiento: 'Mantenimiento', administracion: 'Administración', admin: 'Administrador' }[me.rol]

  const logout = async () => {
    await api('/auth/logout', { body: {} }).catch(() => {})
    nav('/', { replace: true })
    qc.setQueryData(['/me'], null)
    qc.removeQueries({ predicate: (q) => q.queryKey[0] !== '/me' })
  }

  return (
    <div className="min-h-screen md:flex">
      <header className="flex items-center justify-between bg-slate-900 p-3 text-white md:hidden">
        <span className="font-bold">SST Almacén</span>
        <button onClick={() => setOpen(!open)}><Menu /></button>
      </header>
      <aside className={`${open ? 'block' : 'hidden'} bg-slate-900 text-slate-200 md:sticky md:top-0 md:block md:h-screen md:w-60 md:shrink-0`}>
        <div className="hidden p-4 text-lg font-bold text-white md:block">SST Almacén</div>
        <div className="border-b border-slate-700 px-4 pb-3 pt-2 text-sm">
          <span className="rounded bg-blue-600 px-1.5 py-0.5 text-xs text-white">{rolLabel}</span>
          <div className="mt-1 font-semibold text-white">{me.nombre}</div>
          <div className="text-xs text-slate-400">{me.puesto}</div>
          <div className="text-xs text-slate-400">DNI {me.dni}</div>
        </div>
        <nav className="p-2">
          {items.map(([to, l, Icon, n]) => (
            <NavLink key={to} to={to} onClick={() => setOpen(false)}
              className={({ isActive }) => `flex items-center gap-2 rounded px-3 py-2 text-sm ${isActive ? 'bg-slate-700 text-white' : 'hover:bg-slate-800'}`}>
              <Icon size={16} /> <span className="flex-1">{l}</span>
              {!!n && <span className="rounded-full bg-amber-500 px-1.5 text-xs text-white">{n}</span>}
            </NavLink>
          ))}
          <button onClick={logout} className="mt-2 flex w-full cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm hover:bg-slate-800"><LogOut size={16} /> Salir</button>
        </nav>
      </aside>
      <main className="flex-1"><Outlet /></main>
      <Toast />
    </div>
  )
}
