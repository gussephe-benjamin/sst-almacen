// Reemplazo de src/api/client.ts para la maqueta: responde con datos de demostración guardados
// en fixtures.json, sin servidor. Los cambios (crear, aprobar, etc.) responden bien pero no se guardan.
import fixtures from './fixtures.json'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type Row = Record<string, any>

export type Me = {
  usuario_id: number
  dni: string
  nombre: string
  puesto_id: number
  puesto: string
  rol: 'trabajador' | 'sst' | 'mantenimiento' | 'administracion' | 'calidad' | 'admin'
}

export class ApiError extends Error {
  code: string
  status: number
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

type Rol = Me['rol']
const FX = fixtures as unknown as Record<string, Record<string, unknown>> & { _files: Record<string, string> }
export const ROLES: Rol[] = ['trabajador', 'sst', 'mantenimiento', 'administracion', 'calidad', 'admin']

// El rol activo vive en la dirección (?rol=sst): sobrevive a recargas y se puede compartir.
const leer = () => new URLSearchParams(location.search).get('rol') as Rol | null
let rol: Rol | null = ROLES.includes(leer() as Rol) ? leer() : null
export function setRol(r: Rol | null) {
  rol = r
  const u = new URL(location.href)
  if (r) u.searchParams.set('rol', r); else u.searchParams.delete('rol')
  history.replaceState(history.state, '', u)
}
export const meDe = (r: Rol) => FX[r]['/me'] as Me

const espera = () => new Promise((ok) => setTimeout(ok, 120))

// Filtros que la API resuelve con parámetros: aquí se aplican sobre la lista completa.
function filtrar(path: string, datos: Record<string, unknown>): unknown {
  const [base, qs = ''] = path.split('?')
  const p = new URLSearchParams(qs)
  if (base === '/hallazgos') {
    const q = (p.get('q') ?? '').toLowerCase()
    return (datos['/hallazgos?'] as Row[]).filter((h) =>
      (!p.get('estado') || h.estado === p.get('estado')) && (!p.get('riesgo') || h.nivel_riesgo === p.get('riesgo')) &&
      (!Number(p.get('ubicacion')) || h.ubicacion_id === Number(p.get('ubicacion'))) &&
      (!Number(p.get('responsable')) || h.area_responsable_id === Number(p.get('responsable'))) &&
      (!q || `${h.descripcion} ${h.detalle}`.toLowerCase().includes(q)))
  }
  if (base === '/recursos') {
    const c = p.get('categoria') ?? ''
    return (datos['/recursos?categoria='] as Row[]).filter((r) => !c || r.categoria === c)
  }
  return undefined
}

export async function api<T = Row>(path: string, opts: { method?: string; body?: unknown } = {}): Promise<T> {
  await espera()
  const method = opts.method ?? (opts.body !== undefined ? 'POST' : 'GET')
  const body = (opts.body ?? {}) as Row
  if (path === '/auth/login') {
    const r = ROLES.find((x) => meDe(x).dni === body.dni) ?? 'sst'
    setRol(r)
    return meDe(r) as T
  }
  if (path === '/auth/logout') { setRol(null); return {} as T }
  if (path === '/auth/register') return {} as T
  if (!rol) throw new ApiError(401, 'no_autenticado', 'Ingresa para continuar.')
  if (method !== 'GET') {
    if (path.endsWith('/password')) return { password: 'Contraseña de ejemplo' } as T
    return { id: 0 } as T
  }
  const datos = FX[rol]
  if (path in datos) return datos[path] as T
  if (path.startsWith('/admin/usuarios/') && path.endsWith('/password')) return { password: 'Ejemplo2026' } as T
  const f = filtrar(path, datos)
  if (f !== undefined) return f as T
  if (path.includes('?') || /^\/[a-z-]+$/.test(path)) return [] as T
  throw new ApiError(404, 'no_encontrado', 'Esta pantalla no tiene datos en la maqueta.')
}

const locales: Record<string, string> = {}
export async function upload(file: File): Promise<string> {
  await espera()
  const id = `local-${Object.keys(locales).length + 1}`
  locales[id] = URL.createObjectURL(file)
  return id
}

const SIN_FOTO = 'data:image/svg+xml,' + encodeURIComponent(
  '<svg xmlns="http://www.w3.org/2000/svg" width="640" height="420"><rect width="100%" height="100%" fill="#e2e8f0"/>' +
  '<text x="50%" y="50%" text-anchor="middle" font-family="sans-serif" font-size="22" fill="#64748b">Foto de ejemplo</text></svg>')

export const fileUrl = (id: string) => locales[id] ?? FX._files[id] ?? SIN_FOTO
