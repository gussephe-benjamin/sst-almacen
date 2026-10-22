import { Navigate, Route, Routes } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, type Me } from './api/client'
import { Toast } from './components/ui'
import { Layout } from './layouts/Layout'
import { Login } from './features/auth'
import { EppDetalle, EppSst, MisSolicitudes, SolicitarEpp } from './features/epp'
import { Inventario } from './features/inventario'
import { HallazgoDetalle, Hallazgos } from './features/hallazgos'
import { TareaDetalle, Tareas } from './features/tareas'
import { Biblioteca, Dashboard, Internas, Notificaciones, Puestos } from './features/otros'

const HOME: Record<Me['rol'], string> = { trabajador: '/solicitar', sst: '/dashboard', mantenimiento: '/tareas', administracion: '/tareas' }

export default function App() {
  const me = useQuery<Me | null>({ queryKey: ['/me'], queryFn: () => api<Me>('/me').catch(() => null), staleTime: Infinity })
  if (me.isLoading) return <p className="muted p-6">Cargando…</p>
  if (!me.data) return <><Login /><Toast /></>
  const m = me.data
  const sst = m.rol === 'sst'
  const area = m.rol === 'mantenimiento' || m.rol === 'administracion'
  return (
    <Routes>
      <Route element={<Layout me={m} />}>
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
        <Route path="/biblioteca" element={<Biblioteca me={m} />} />
        <Route path="/notificaciones" element={<Notificaciones />} />
        <Route path="*" element={<Navigate to={HOME[m.rol]} replace />} />
      </Route>
    </Routes>
  )
}
