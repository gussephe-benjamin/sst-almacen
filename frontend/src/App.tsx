import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, type Me } from './api/client'
import { Toast, toast } from './components/ui'
import { Layout } from './layouts/Layout'
import { Login } from './features/auth'
import { Cuentas } from './features/admin'
import { EppDetalle, EppSst, MisSolicitudes, SolicitarEpp } from './features/epp'
import { Inventario } from './features/inventario'
import { HallazgoDetalle, Hallazgos } from './features/hallazgos'
import { TareaDetalle, Tareas } from './features/tareas'
import { Biblioteca, Dashboard, Internas, Notificaciones, Puestos } from './features/otros'

export const HOME: Record<Me['rol'], string> = { trabajador: '/solicitar', sst: '/dashboard', mantenimiento: '/tareas', administracion: '/tareas', admin: '/cuentas' }

/** Sin sesión válida devuelve null; un fallo de red se reintenta en lugar de cerrar la sesión. */
async function loadMe(): Promise<Me | null> {
  try {
    return await api<Me>('/me')
  } catch (e) {
    if (e instanceof ApiError && (e.status === 401 || e.status === 403)) return null
    throw e
  }
}

export default function App() {
  const qc = useQueryClient()
  const me = useQuery<Me | null>({ queryKey: ['/me'], queryFn: loadMe, staleTime: Infinity, retry: 2 })

  useEffect(() => {
    const onExpired = (e: Event) => {
      if (!qc.getQueryData(['/me'])) return
      qc.setQueryData(['/me'], null)
      qc.removeQueries({ predicate: (q) => q.queryKey[0] !== '/me' })
      toast((e as CustomEvent).detail || 'Tu sesión terminó. Vuelve a ingresar.', true)
    }
    window.addEventListener('sesion-expirada', onExpired)
    return () => window.removeEventListener('sesion-expirada', onExpired)
  }, [qc])

  if (me.isLoading) return <p className="muted p-6">Cargando…</p>
  if (me.isError) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 p-6 text-center">
        <p>No se pudo conectar con el servidor.</p>
        <button className="btn btn-primary" onClick={() => me.refetch()}>Reintentar</button>
      </div>
    )
  }
  if (!me.data) return <><Login /><Toast /></>
  const m = me.data
  const sst = m.rol === 'sst'
  const area = m.rol === 'mantenimiento' || m.rol === 'administracion'
  return (
    <Routes>
      <Route element={<Layout me={m} />}>
        {m.rol === 'admin' && <Route path="/cuentas" element={<Cuentas me={m} />} />}
        {m.rol === 'trabajador' && <>
          <Route path="/solicitar" element={<SolicitarEpp />} />
          <Route path="/mis-solicitudes" element={<MisSolicitudes />} />
        </>}
        {(m.rol === 'trabajador' || sst) && <Route path="/epp/:id" element={<EppDetalle me={m} />} />}
        {sst && <>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/hallazgos" element={<Hallazgos />} />
          <Route path="/hallazgos/:id" element={<HallazgoDetalle />} />
          <Route path="/epp" element={<EppSst />} />
          <Route path="/inventario" element={<Inventario />} />
          <Route path="/internas" element={<Internas />} />
          <Route path="/puestos" element={<Puestos />} />
        </>}
        {(sst || area) && <>
          <Route path="/tareas" element={<Tareas me={m} />} />
          <Route path="/tareas/:id" element={<TareaDetalle me={m} />} />
        </>}
        {m.rol !== 'admin' && <Route path="/biblioteca" element={<Biblioteca me={m} />} />}
        <Route path="/notificaciones" element={<Notificaciones />} />
        <Route path="*" element={<Navigate to={HOME[m.rol]} replace />} />
      </Route>
    </Routes>
  )
}
