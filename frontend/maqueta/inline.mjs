// Junta el HTML, el JS y el CSS de maqueta/dist en un solo archivo que se abre sin servidor.
import { readFileSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const aqui = path.dirname(fileURLToPath(import.meta.url))
const dist = path.join(aqui, 'dist')
const salida = path.resolve(aqui, '../../docs/maqueta-producto.html')

let html = readFileSync(path.join(dist, 'index.html'), 'utf8')
const leer = (ref) => readFileSync(path.join(dist, ref.replace(/^\.\//, '')), 'utf8')

html = html.replace(/<link rel="stylesheet"[^>]*href="([^"]+)"[^>]*>/g, (_, ref) => `<style>${leer(ref)}</style>`)
let scripts = ''
html = html.replace(/<script type="module"[^>]*src="([^"]+)"[^>]*><\/script>/g, (_, ref) => {
  // `</script` dentro del código cerraría la etiqueta antes de tiempo.
  scripts += `<script type="module">${leer(ref).replace(/<\/script/gi, '<\\/script')}</script>`
  return ''
})
// El módulo va al final del body para que #root ya exista.
html = html.replace('</body>', `${scripts}\n</body>`)
writeFileSync(salida, html)
console.log(`Maqueta lista: ${path.relative(process.cwd(), salida)} (${Math.round(html.length / 1024)} KB)`)
