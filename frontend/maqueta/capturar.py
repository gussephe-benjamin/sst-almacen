import os, urllib.parse, json, re, io, base64, urllib.request, http.cookiejar, datetime
from PIL import Image, ImageDraw, ImageFont

# Captura lo que ve cada rol en una API con datos de demostración (ver maqueta/README.md).
BASE = os.environ.get('MAQUETA_API', 'http://localhost:8091/api')
PW = os.environ['MAQUETA_PW']  # contraseña de ACCOUNTS_ADMIN y ACCOUNTS_CALIDAD de la API temporal


def cliente(dni, pw):
    cj = http.cookiejar.CookieJar()
    op = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cj))

    def call(method, path, body=None, raw=False, ctype='application/json'):
        data = body if raw else (json.dumps(body).encode() if body is not None else None)
        req = urllib.request.Request(BASE + path, method=method, data=data,
                                     headers={'Content-Type': ctype, 'Origin': 'http://localhost:8091'})
        try:
            r = op.open(req); code = r.status; out = r.read()
        except urllib.error.HTTPError as e:
            code = e.code; out = e.read()
        if code >= 400:
            print('FALLA', method, path, code, out[:200])
            return None
        if raw and method == 'GET':
            return out
        return json.loads(out) if out else None

    call('POST', '/auth/login', {'dni': dni, 'password': pw})
    return call


def font(n):
    for f in ['C:/Windows/Fonts/segoeuib.ttf', 'C:/Windows/Fonts/arialbd.ttf']:
        try:
            return ImageFont.truetype(f, n)
        except OSError:
            pass
    return ImageFont.load_default()


def afiche(w, h, fondo, titulo, sub):
    im = Image.new('RGB', (w, h), fondo)
    d = ImageDraw.Draw(im)
    d.rectangle([0, 0, w, int(h * .18)], fill=(250, 204, 21))
    d.text((int(w * .06), int(h * .05)), 'SEGURIDAD', font=font(int(h * .07)), fill=(30, 30, 30))
    y = int(h * .3)
    for linea in titulo.split('\n'):
        d.text((int(w * .06), y), linea, font=font(int(min(w, h) * .11)), fill='white')
        y += int(min(w, h) * .14)
    d.text((int(w * .06), y + 20), sub, font=font(int(min(w, h) * .045)), fill=(226, 232, 240))
    d.ellipse([w - int(w * .3), h - int(w * .3), w + int(w * .1), h + int(w * .1)], outline=(250, 204, 21), width=10)
    b = io.BytesIO()
    im.save(b, 'PNG')
    return b.getvalue()


def subir(call, png):
    bd = '----maqueta'
    body = (f'--{bd}\r\nContent-Disposition: form-data; name="file"; filename="a.png"\r\n'
            f'Content-Type: image/png\r\n\r\n').encode() + png + f'\r\n--{bd}--\r\n'.encode()
    return call('POST', '/files', body, raw=True, ctype=f'multipart/form-data; boundary={bd}')['id']


C = {
    'trabajador': cliente('71234567', 'demo1234'), 'sst': cliente('45678912', 'demo1234'),
    'mantenimiento': cliente('43219876', 'demo1234'), 'administracion': cliente('46781234', 'demo1234'),
    'calidad': cliente('47812345', PW), 'admin': cliente('10000001', PW),
}
S = C['sst']

# --- Enriquecer la demo con lo agregado en las últimas versiones ---
for tipo, cod, tit, res, w, h, color, linea in [
    ('afiche', 'AF-010', 'Usa tu EPP completo', 'Casco, chaleco, botas y guantes en toda la zona de carga.', 600, 850, (29, 78, 216), 'Usa tu EPP\ncompleto'),
    ('flyer', 'FL-004', 'Orden y limpieza en pasillos', 'Pasillos libres de cajas y parihuelas.', 900, 520, (5, 150, 105), 'Pasillos\nlibres'),
    ('afiche', 'AF-011', 'Levanta la carga con las piernas', 'Espalda recta, carga pegada al cuerpo.', 600, 850, (185, 28, 28), 'Levanta\nbien'),
    ('instructivo', 'IN-007', 'Uso del extintor PQS', 'Hala, apunta, aprieta y barre.', 900, 600, (194, 65, 12), 'Extintor\nPQS'),
]:
    img = subir(S, afiche(w, h, color, linea, res[:42]))
    S('POST', '/biblioteca', {'tipo': tipo, 'codigo': cod, 'titulo': tit, 'resumen': res,
                              'cuerpo': res + '\n\nConsulta con SST ante cualquier duda.', 'imagen_id': img})

cat = S('GET', '/catalogos')
cid = {c['nombre']: c['id'] for c in cat['categorias']}
if 'Extintores' in cid:
    S('POST', '/recursos', {'nombre': 'Extintor 6 kg', 'categoria_id': cid['Extintores'], 'tallas': 'extintor',
                            'variantes': {'PQS': 8, 'CO2': 1, 'Agua presurizada': 2},
                            'minimos': {'PQS': 3, 'CO2': 2, 'Agua presurizada': 1}})
rs = S('GET', '/recursos?categoria=')
botas = next((r for r in rs if r['nombre'] == 'Botas de seguridad'), None)
if botas and botas['tallas']:
    S('POST', f"/recursos/{botas['id']}/movimientos", {'tipo': 'ingreso', 'nota': 'Compra de noviembre', 'items': [
        {'variante': '40', 'cantidad': 4}, {'variante': '41', 'cantidad': 3}, {'variante': '42', 'cantidad': 1}]})
    S('PATCH', f"/recursos/{botas['id']}", {'minimos': {'40': 2, '41': 2, '42': 2, '43': 2}})

M = C['mantenimiento']
foto = subir(M, afiche(640, 480, (71, 85, 105), 'Guantes\nrotos', 'Foto de muestra'))
guantes = next(r for r in rs if r['nombre'] == 'Guantes de seguridad')
talla = {'letras': 'L', 'calzado': '41'}.get(guantes['tallas'] or '', '')
M('POST', '/epp', {'recurso_id': guantes['id'], 'motivo': 'Desgaste', 'talla': talla, 'foto_id': foto,
                   'descripcion': 'Se rompieron al mover parihuelas.'})

# --- Capturar lo que ve cada rol ---
fx = {}


def get(rol, path):
    v = C[rol]('GET', path)
    fx.setdefault(rol, {})[path] = v
    return v or []


for rol in C:
    get(rol, '/me'); get(rol, '/notificaciones'); get(rol, '/areas')
    if rol != 'admin':
        get(rol, '/catalogos')
        for t in ['afiche', 'flyer', 'procedimiento', 'instructivo']:
            get(rol, f'/biblioteca?tipo={t}')
        for b in get(rol, '/biblioteca?tipo='):
            get(rol, f"/biblioteca/{b['id']}")
    if rol in ('trabajador', 'mantenimiento', 'administracion', 'calidad'):
        for e in get(rol, '/epp/mias'):
            get(rol, f"/epp/{e['id']}")
    if rol in ('sst', 'mantenimiento', 'administracion', 'calidad'):
        for t in get(rol, '/tareas'):
            get(rol, f"/tareas/{t['id']}")
    if rol == 'sst':
        get(rol, '/dashboard'); get(rol, '/tareas-internas')
        for p in get(rol, '/puestos'):
            get(rol, f"/puestos/{p['id']}/historial")
        for est in ['enviada', 'aprobada', 'en_espera', 'entregada', 'rechazada', '']:
            for e in get(rol, f'/epp?estado={est}'):
                get(rol, f"/epp/{e['id']}")
        for c in [''] + [c['nombre'] for c in cat['categorias']]:
            get(rol, '/recursos?categoria=' + urllib.parse.quote(c))
        for h in get(rol, '/hallazgos?'):
            get(rol, f"/hallazgos/{h['id']}")
    if rol == 'admin':
        cu = C[rol]('GET', '/admin/cuentas')
        fx[rol]['/admin/cuentas'] = cu
        for p in cu['puestos']:
            get(rol, f"/admin/puestos/{p['id']}/historial")

# Archivos referenciados (fotos e imágenes), como data URI
texto = json.dumps(fx)
ids = set(re.findall(r'"([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"', texto))
files = {}
for i in ids:
    raw = S('GET', f'/files/{i}', raw=True)
    if raw and raw[:4] == b'\x89PNG':
        files[i] = 'data:image/png;base64,' + base64.b64encode(raw).decode()
    elif raw and raw[:2] == b'\xff\xd8':
        files[i] = 'data:image/jpeg;base64,' + base64.b64encode(raw).decode()
fx['_files'] = files
json.dump(fx, open(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'fixtures.json'), 'w', encoding='utf-8'), ensure_ascii=False)
print({k: len(v) for k, v in fx.items()})
print('hallazgo:', list(fx['sst']['/hallazgos?'][0].keys()))
print('epp:', list(fx['sst']['/epp?estado='][0].keys()))
