// Entrada de la maqueta: la misma App, con HashRouter (funciona abriendo el archivo sin servidor)
// y una barra flotante para ver la plataforma como cada rol.
import { StrictMode, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { HashRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query'
import './estilos.css'
import App, { HOME } from '../src/App'
import { ROLES, meDe, type Me } from './client'

const qc = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false, staleTime: Infinity } } })

const NOMBRE: Record<Me['rol'], string> = {
  trabajador: 'Trabajador', sst: 'SST', mantenimiento: 'Mantenimiento', administracion: 'Administración', calidad: 'Calidad', admin: 'Administrador',
}

// Cambiar de rol recarga la maqueta en la pantalla de inicio de ese rol.
function ver(r: Me['rol'] | '') {
  const u = new URL(location.href)
  if (r) u.searchParams.set('rol', r); else u.searchParams.delete('rol')
  u.hash = r ? HOME[r] : '/'
  location.href = u.toString() // cambia ?rol=, así que el navegador recarga la página
}

// Las descargas y los enlaces a archivos no existen sin servidor.
document.addEventListener('click', (e) => {
  const a = (e.target as HTMLElement).closest('a')
  const href = a?.getAttribute('href') ?? ''
  if (href.startsWith('/api/') || href.startsWith('data:')) {
    e.preventDefault()
    if (href.startsWith('/api/hallazgos/export')) alert('En la maqueta no se descargan archivos. En la plataforma, este botón descarga el reporte.')
  }
}, true)

function Barra() {
  // En celular empieza cerrada para no tapar la pantalla.
  const [abierta, setAbierta] = useState(window.innerWidth >= 640)
  // Sigue la sesión de la App: cambia también al ingresar o salir desde la propia plataforma.
  const me = useQuery<Me | null>({ queryKey: ['/me'], enabled: false })
  const actual = me.data?.rol ?? ''
  if (!abierta) {
    return <button onClick={() => setAbierta(true)} className="fixed bottom-4 right-4 z-[100] rounded-full bg-amber-400 px-4 py-2 text-sm font-semibold text-slate-900 shadow-lg">Maqueta</button>
  }
  return (
    <div className="fixed bottom-4 right-4 z-[100] w-64 rounded-xl border border-amber-300 bg-white p-3 text-sm shadow-xl">
      <div className="mb-2 flex items-center justify-between">
        <strong>Maqueta · SST Almacén</strong>
        <button onClick={() => setAbierta(false)} className="text-slate-400 hover:text-slate-700" aria-label="Ocultar">✕</button>
      </div>
      <label className="mb-1 block text-xs text-slate-500">Ver como</label>
      <select className="inp mb-2 w-full" value={actual} onChange={(e) => ver(e.target.value as Me['rol'] | '')}>
        <option value="">Sin sesión (ingreso)</option>
        {ROLES.map((r) => <option key={r} value={r}>{NOMBRE[r]} · {meDe(r).nombre}</option>)}
      </select>
      <p className="text-xs text-slate-500">Datos de demostración. Los botones funcionan, pero los cambios no se guardan. Para la vista de celular, achica la ventana.</p>
    </div>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={qc}>
      <HashRouter>
        <App />
      </HashRouter>
      <Barra />
    </QueryClientProvider>
  </StrictMode>,
)
