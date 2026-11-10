// Compila la maqueta: la App real, con src/api/client.ts reemplazado por maqueta/client.ts.
// Uso: npm run maqueta  → genera ../docs/maqueta-producto.html (un solo archivo).
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { defineConfig, type Plugin } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const aqui = path.dirname(fileURLToPath(import.meta.url))
const clienteReal = path.resolve(aqui, '../src/api/client')

const clienteMaqueta = (): Plugin => ({
  name: 'cliente-maqueta',
  enforce: 'pre',
  async resolveId(source, importer) {
    if (!importer || !source.endsWith('api/client')) return null
    if (path.resolve(path.dirname(importer), source) === clienteReal) return path.resolve(aqui, 'client.ts')
    return null
  },
})

export default defineConfig({
  root: aqui,
  base: './',
  plugins: [clienteMaqueta(), react(), tailwindcss()],
  build: { outDir: path.resolve(aqui, 'dist'), emptyOutDir: true, assetsInlineLimit: 100_000_000, cssCodeSplit: false },
})
