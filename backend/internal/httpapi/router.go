package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"sst-almacen/internal/service"
	"sst-almacen/internal/storage"
)

type API struct {
	S            *service.Svc
	Files        *storage.Store
	Secret       string
	CookieSecure bool
	StaticDir    string
	limiter      *loginLimiter
}

const maxUpload = 5 << 20

func (a *API) Router() http.Handler {
	a.limiter = newLoginLimiter()
	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer, securityHeaders)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) { ok(w, nil) })
		r.Get("/areas", func(w http.ResponseWriter, r *http.Request) {
			v, err := a.S.AreasPublicas(r.Context())
			reply(w, v, err)
		})
		r.Post("/auth/register", a.register)
		r.Post("/auth/login", a.login)
		r.Post("/auth/logout", func(w http.ResponseWriter, _ *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: cookieName, Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteLaxMode})
			ok(w, nil)
		})

		r.Group(func(r chi.Router) {
			r.Use(a.auth)
			r.Get("/me", func(w http.ResponseWriter, r *http.Request) { reply(w, actor(r), nil) })
			r.Post("/me/password", func(w http.ResponseWriter, r *http.Request) {
				var in struct{ Actual, Nueva string }
				if err := decode(r, &in); err != nil {
					writeErr(w, err)
					return
				}
				ok(w, a.S.CambiarPassword(r.Context(), actor(r), in.Actual, in.Nueva))
			})
			r.Get("/catalogos", func(w http.ResponseWriter, r *http.Request) { v, err := a.S.Catalogos(r.Context()); reply(w, v, err) })
			r.Post("/files", a.upload)
			r.Get("/files/{id}", a.file)

			// Notificaciones (todos)
			r.Get("/notificaciones", func(w http.ResponseWriter, r *http.Request) {
				v, err := a.S.Notificaciones(r.Context(), actor(r))
				reply(w, v, err)
			})
			r.Post("/notificaciones/leer-todas", func(w http.ResponseWriter, r *http.Request) { ok(w, a.S.LeerNotificacion(r.Context(), actor(r), 0)) })
			r.Post("/notificaciones/{id}/leer", func(w http.ResponseWriter, r *http.Request) {
				id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
				ok(w, a.S.LeerNotificacion(r.Context(), actor(r), id))
			})

			// Biblioteca
			r.Get("/biblioteca", func(w http.ResponseWriter, r *http.Request) {
				v, err := a.S.Biblioteca(r.Context(), r.URL.Query().Get("tipo"))
				reply(w, v, err)
			})
			r.Get("/biblioteca/{id}", func(w http.ResponseWriter, r *http.Request) {
				v, err := a.S.BiblioDetalle(r.Context(), idParam(r, "id"))
				reply(w, v, err)
			})

			// EPP
			r.With(RequireRole("trabajador")).Get("/epp/mias", func(w http.ResponseWriter, r *http.Request) {
				v, err := a.S.EppMias(r.Context(), actor(r))
				reply(w, v, err)
			})
			r.With(RequireRole("trabajador")).Post("/epp", func(w http.ResponseWriter, r *http.Request) {
				var in service.EppIn
				if err := decode(r, &in); err != nil {
					writeErr(w, err)
					return
				}
				id, err := a.S.EppCrear(r.Context(), actor(r), in)
				reply(w, map[string]int{"id": id}, err)
			})
			r.With(RequireRole("trabajador", "sst")).Get("/epp/{id}", func(w http.ResponseWriter, r *http.Request) {
				v, err := a.S.EppDetalle(r.Context(), actor(r), idParam(r, "id"))
				reply(w, v, err)
			})

			// Tareas (SST y áreas; la propiedad se valida en el servicio)
			r.With(RequireRole("sst", "mantenimiento", "administracion")).Route("/tareas", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.Tareas(r.Context(), actor(r))
					reply(w, v, err)
				})
				r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.TareaDetalle(r.Context(), actor(r), idParam(r, "id"))
					reply(w, v, err)
				})
				r.Post("/{id}/aceptar", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.TareaAceptar(r.Context(), actor(r), idParam(r, "id")))
				})
				r.Post("/{id}/modificacion", func(w http.ResponseWriter, r *http.Request) {
					var in service.ModIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.TareaPedirModificacion(r.Context(), actor(r), idParam(r, "id"), in))
				})
				r.Post("/{id}/pasos/{orden}/completar", func(w http.ResponseWriter, r *http.Request) {
					var in service.PasoIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.PasoCompletar(r.Context(), actor(r), idParam(r, "id"), idParam(r, "orden"), in, true))
				})
				r.Post("/{id}/pasos/{orden}/reabrir", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.PasoCompletar(r.Context(), actor(r), idParam(r, "id"), idParam(r, "orden"), service.PasoIn{}, false))
				})
				r.Post("/{id}/terminar", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.TareaTerminar(r.Context(), actor(r), idParam(r, "id")))
				})

				r.Group(func(r chi.Router) {
					r.Use(RequireRole("sst"))
					r.Post("/{id}/modificacion/aprobar", func(w http.ResponseWriter, r *http.Request) {
						ok(w, a.S.TareaResponderModificacion(r.Context(), actor(r), idParam(r, "id"), true, ""))
					})
					r.Post("/{id}/modificacion/rechazar", func(w http.ResponseWriter, r *http.Request) {
						var in service.ModIn
						decode(r, &in)
						ok(w, a.S.TareaResponderModificacion(r.Context(), actor(r), idParam(r, "id"), false, in.Respuesta))
					})
					r.Post("/{id}/cerrar", func(w http.ResponseWriter, r *http.Request) {
						ok(w, a.S.TareaCerrar(r.Context(), actor(r), idParam(r, "id")))
					})
					r.Post("/{id}/devolver", func(w http.ResponseWriter, r *http.Request) {
						var in service.DevolverIn
						if err := decode(r, &in); err != nil {
							writeErr(w, err)
							return
						}
						ok(w, a.S.TareaDevolver(r.Context(), actor(r), idParam(r, "id"), in))
					})
				})
			})

			// Solo admin: gestión de cuentas
			r.Route("/admin", func(r chi.Router) {
				r.Use(RequireRole("admin"))
				r.Get("/cuentas", func(w http.ResponseWriter, r *http.Request) {
					p, err := a.S.Puestos(r.Context(), true)
					if err != nil {
						writeErr(w, err)
						return
					}
					sp, err := a.S.Pendientes(r.Context())
					reply(w, map[string]any{"puestos": p, "sin_puesto": sp}, err)
				})
				r.Post("/puestos", func(w http.ResponseWriter, r *http.Request) {
					var in service.PuestoIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					id, err := a.S.CrearPuesto(r.Context(), actor(r), in)
					reply(w, map[string]int{"id": id}, err)
				})
				r.Delete("/usuarios/{id}", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.EliminarUsuario(r.Context(), actor(r), idParam(r, "id")))
				})
				r.Post("/cuentas", func(w http.ResponseWriter, r *http.Request) {
					var in service.CuentaIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					id, err := a.S.CrearCuenta(r.Context(), actor(r), in)
					reply(w, map[string]int{"id": id}, err)
				})
				r.Post("/puestos/{id}/asignar", func(w http.ResponseWriter, r *http.Request) {
					var in struct {
						UsuarioID int `json:"usuario_id"`
					}
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.Asignar(r.Context(), actor(r), idParam(r, "id"), in.UsuarioID))
				})
				r.Post("/puestos/{id}/liberar", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.LiberarPuesto(r.Context(), actor(r), idParam(r, "id")))
				})
				r.Get("/puestos/{id}/historial", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.PuestoHistorial(r.Context(), idParam(r, "id"))
					reply(w, v, err)
				})
				r.Get("/usuarios/{id}/password", func(w http.ResponseWriter, r *http.Request) {
					pw, err := a.S.VerPassword(r.Context(), actor(r), idParam(r, "id"))
					reply(w, map[string]string{"password": pw}, err)
				})
				r.Post("/usuarios/{id}/password", func(w http.ResponseWriter, r *http.Request) {
					var in struct {
						Password string `json:"password"`
					}
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.ResetPassword(r.Context(), actor(r), idParam(r, "id"), in.Password))
				})
			})

			// Solo SST
			r.Group(func(r chi.Router) {
				r.Use(RequireRole("sst"))
				r.Get("/dashboard", func(w http.ResponseWriter, r *http.Request) { v, err := a.S.Dashboard(r.Context()); reply(w, v, err) })

				r.Get("/epp", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.EppLista(r.Context(), r.URL.Query().Get("estado"))
					reply(w, v, err)
				})
				for _, acc := range []string{"aprobar", "espera", "rechazar", "entregar"} {
					acc := acc
					r.Post("/epp/{id}/"+acc, func(w http.ResponseWriter, r *http.Request) {
						var in service.EppAccionIn
						decode(r, &in)
						ok(w, a.S.EppAccion(r.Context(), actor(r), idParam(r, "id"), acc, in))
					})
				}

				r.Get("/recursos", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.Recursos(r.Context(), r.URL.Query().Get("categoria"))
					reply(w, v, err)
				})
				r.Post("/recursos", func(w http.ResponseWriter, r *http.Request) {
					var in service.RecursoIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					id, err := a.S.RecursoCrear(r.Context(), actor(r), in)
					reply(w, map[string]int{"id": id}, err)
				})
				r.Patch("/recursos/{id}", func(w http.ResponseWriter, r *http.Request) {
					var in struct {
						StockMinimo int `json:"stock_minimo"`
					}
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.RecursoMinimo(r.Context(), actor(r), idParam(r, "id"), in.StockMinimo))
				})
				r.Post("/recursos/{id}/movimientos", func(w http.ResponseWriter, r *http.Request) {
					var in service.MovIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.Movimiento(r.Context(), actor(r), idParam(r, "id"), in))
				})

				r.Get("/hallazgos", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.Hallazgos(r.Context(), filtro(r))
					reply(w, v, err)
				})
				r.Get("/hallazgos/export.xlsx", a.exportXLSX)
				r.Get("/hallazgos/export.pdf", a.exportPDF)
				r.Post("/hallazgos", func(w http.ResponseWriter, r *http.Request) {
					var in service.HallazgoIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					id, err := a.S.HallazgoCrear(r.Context(), actor(r), in)
					reply(w, map[string]int{"id": id}, err)
				})
				r.Get("/hallazgos/{id}", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.HallazgoDetalle(r.Context(), idParam(r, "id"))
					reply(w, v, err)
				})
				r.Post("/hallazgos/{id}/descartar", func(w http.ResponseWriter, r *http.Request) {
					var in struct {
						Motivo string `json:"motivo"`
					}
					decode(r, &in)
					ok(w, a.S.HallazgoDescartar(r.Context(), actor(r), idParam(r, "id"), in.Motivo))
				})
				r.Post("/hallazgos/{id}/cerrar", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.HallazgoCerrar(r.Context(), actor(r), idParam(r, "id")))
				})
				r.Post("/hallazgos/{id}/tareas", func(w http.ResponseWriter, r *http.Request) {
					var in service.TareaIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					id, err := a.S.TareaCrear(r.Context(), actor(r), idParam(r, "id"), in)
					reply(w, map[string]int{"id": id}, err)
				})

				r.Get("/tareas-internas", func(w http.ResponseWriter, r *http.Request) { v, err := a.S.Internas(r.Context()); reply(w, v, err) })
				r.Post("/tareas-internas", func(w http.ResponseWriter, r *http.Request) {
					var in service.InternaIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.InternaGuardar(r.Context(), actor(r), 0, in))
				})
				r.Patch("/tareas-internas/{id}", func(w http.ResponseWriter, r *http.Request) {
					var in service.InternaIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.InternaGuardar(r.Context(), actor(r), idParam(r, "id"), in))
				})

				saveBiblio := func(w http.ResponseWriter, r *http.Request) {
					var in service.BiblioIn
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					id, err := a.S.BiblioGuardar(r.Context(), actor(r), idParam(r, "id"), in)
					reply(w, map[string]int{"id": id}, err)
				}
				r.Post("/biblioteca", saveBiblio)
				r.Put("/biblioteca/{id}", saveBiblio)
				r.Delete("/biblioteca/{id}", func(w http.ResponseWriter, r *http.Request) {
					ok(w, a.S.BiblioEliminar(r.Context(), actor(r), idParam(r, "id")))
				})

				r.Get("/puestos", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.Puestos(r.Context(), false)
					reply(w, v, err)
				})
				r.Get("/usuarios/pendientes", func(w http.ResponseWriter, r *http.Request) { v, err := a.S.Pendientes(r.Context()); reply(w, v, err) })
				r.Post("/puestos/{id}/asignar", func(w http.ResponseWriter, r *http.Request) {
					var in struct {
						UsuarioID int `json:"usuario_id"`
					}
					if err := decode(r, &in); err != nil {
						writeErr(w, err)
						return
					}
					ok(w, a.S.Asignar(r.Context(), actor(r), idParam(r, "id"), in.UsuarioID))
				})
				r.Get("/puestos/{id}/historial", func(w http.ResponseWriter, r *http.Request) {
					v, err := a.S.PuestoHistorial(r.Context(), idParam(r, "id"))
					reply(w, v, err)
				})
			})
		})
	})
	a.serveStatic(r)
	return r
}

// serveStatic sirve el frontend compilado y devuelve index.html para las rutas de la SPA.
func (a *API) serveStatic(r chi.Router) {
	index := filepath.Join(a.StaticDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) { writeErr(w, service.ErrNotFound) })
		return
	}
	fs := http.FileServer(http.Dir(a.StaticDir))
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, "/api/") {
			writeErr(w, service.ErrNotFound)
			return
		}
		if p := filepath.Join(a.StaticDir, filepath.Clean(req.URL.Path)); req.URL.Path != "/" {
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				if strings.HasPrefix(req.URL.Path, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fs.ServeHTTP(w, req)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, req, index)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func filtro(r *http.Request) service.HallazgoFiltro {
	q := r.URL.Query()
	ub, _ := strconv.Atoi(q.Get("ubicacion"))
	resp, _ := strconv.Atoi(q.Get("responsable"))
	return service.HallazgoFiltro{Estado: q.Get("estado"), Riesgo: q.Get("riesgo"), Q: q.Get("q"), Ubicacion: ub, Responsable: resp}
}

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var in service.RegisterIn
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	ok(w, a.S.Register(r.Context(), in))
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ DNI, Password string }
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	keys := []string{"dni:" + in.DNI, "ip:" + clientIP(r)}
	if a.limiter.blocked(keys...) {
		writeErr(w, service.Err(http.StatusTooManyRequests, "demasiados_intentos", "Demasiados intentos fallidos. Espera 15 minutos e intenta de nuevo."))
		return
	}
	id, err := a.S.Login(r.Context(), in.DNI, in.Password)
	if err != nil {
		var ae *service.AppError
		if errors.As(err, &ae) && ae.Status == http.StatusUnauthorized {
			a.limiter.fail(keys...)
		}
		writeErr(w, err)
		return
	}
	a.limiter.reset(keys...)
	a.issue(w, id)
	act, err := a.S.LoadActor(r.Context(), id)
	reply(w, act, err)
}

var exts = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}

func (a *API) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+1024)
	f, _, err := r.FormFile("file")
	if err != nil {
		writeErr(w, service.Bad("archivo_invalido", "La imagen no debe superar 5 MB."))
		return
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := f.Read(head)
	mime := http.DetectContentType(head[:n])
	ext, okType := exts[mime]
	if !okType {
		writeErr(w, service.Bad("tipo_invalido", "Solo se permiten imágenes (JPG, PNG, WEBP o GIF)."))
		return
	}
	f.Seek(0, 0)
	name, size, err := a.Files.Save(f, ext)
	if err != nil {
		writeErr(w, err)
		return
	}
	id, err := a.S.GuardarArchivo(r.Context(), actor(r), name, mime, size)
	reply(w, map[string]string{"id": id}, err)
}

func (a *API) file(w http.ResponseWriter, r *http.Request) {
	ruta, mime, err := a.S.PuedeVerArchivo(r.Context(), actor(r), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	if strings.Contains(ruta, "..") {
		writeErr(w, service.ErrNotFound)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, a.Files.Path(filepath.Base(ruta)))
}
