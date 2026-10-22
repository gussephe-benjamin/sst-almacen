// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type Row = Record<string, any>

export type Me = {
  usuario_id: number
  dni: string
  nombre: string
  puesto_id: number
  puesto: string
  rol: 'trabajador' | 'sst' | 'mantenimiento' | 'administracion'
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

export async function api<T = Row>(path: string, opts: { method?: string; body?: unknown } = {}): Promise<T> {
  const res = await fetch('/api' + path, {
    method: opts.method ?? (opts.body !== undefined ? 'POST' : 'GET'),
    headers: opts.body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
    credentials: 'include',
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new ApiError(res.status, data?.error?.code ?? 'error', data?.error?.message ?? 'Ocurrió un error inesperado.')
  }
  return data as T
}

export async function upload(file: File): Promise<string> {
  const fd = new FormData()
  fd.append('file', file)
  const res = await fetch('/api/files', { method: 'POST', body: fd, credentials: 'include' })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, data?.error?.code, data?.error?.message ?? 'No se pudo subir la imagen.')
  return data.id
}

export const fileUrl = (id: string) => `/api/files/${id}`
