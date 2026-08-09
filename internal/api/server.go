// Package api expone las funcionalidades del sistema mediante servicios
// REST usando la biblioteca estándar net/http. Todos los endpoints
// serializan sus respuestas en JSON, cumpliendo con el requisito del
// enunciado de "generación de servicios web con serialización JSON".
//
// La decisión de usar net/http en vez de un framework externo (Gin, Echo)
// se justifica en tres puntos:
//  1. Cero dependencias externas — el proyecto compila con solo la
//     biblioteca estándar de Go.
//  2. Muestra dominio real del lenguaje, no de un framework de moda.
//  3. La superficie de código es más pequeña y más fácil de explicar.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ronnycort/leelibre/internal/catalogo"
	"github.com/ronnycort/leelibre/internal/errores"
	"github.com/ronnycort/leelibre/internal/prestamos"
	"github.com/ronnycort/leelibre/internal/reportes"
	"github.com/ronnycort/leelibre/internal/reservas"
	"github.com/ronnycort/leelibre/internal/usuarios"
)

// Server encapsula el servidor HTTP y las referencias a los subsistemas.
// Tiene la misma estructura que App en la CLI para que los mismos objetos
// del dominio se compartan entre CLI y web sin duplicación.
type Server struct {
	catalogo       *catalogo.Catalogo
	autenticador   *usuarios.Autenticador
	historial      *prestamos.Historial
	gestorReservas *reservas.GestorReservas
}

// NewServer construye el servidor. Recibe los subsistemas ya cargados.
func NewServer(
	cat *catalogo.Catalogo,
	auth *usuarios.Autenticador,
	hist *prestamos.Historial,
	gr *reservas.GestorReservas,
) *Server {
	return &Server{
		catalogo:       cat,
		autenticador:   auth,
		historial:      hist,
		gestorReservas: gr,
	}
}

// Rutas registra todos los endpoints REST en el multiplexor por defecto
// de net/http y retorna el mux configurado. Cada endpoint tiene una
// función handler específica; el patrón es idéntico al de Gin pero
// usando solo la biblioteca estándar.
func (s *Server) Rutas() *http.ServeMux {
	mux := http.NewServeMux()

	// Endpoint 1: GET /api/libros - listar todos los libros
	mux.HandleFunc("GET /api/libros", s.listarLibros)

	// Endpoint 2: GET /api/libros/{id} - obtener un libro específico
	mux.HandleFunc("GET /api/libros/{id}", s.obtenerLibro)

	// Endpoint 3: POST /api/libros - agregar un libro (solo administrador)
	mux.HandleFunc("POST /api/libros", s.conAdmin(s.crearLibro))

	// Endpoint 4: DELETE /api/libros/{id} - eliminar un libro (solo administrador)
	mux.HandleFunc("DELETE /api/libros/{id}", s.conAdmin(s.eliminarLibro))

	// Endpoint 5: PUT /api/libros/{id} - modificar un libro (solo administrador)
	mux.HandleFunc("PUT /api/libros/{id}", s.conAdmin(s.actualizarLibro))

	// Endpoint 5: GET /api/usuarios/{id}/recomendaciones - recomendador
	mux.HandleFunc("GET /api/usuarios/{id}/recomendaciones", s.recomendaciones)

	// Endpoint 6: POST /api/prestamos - registrar un préstamo o reserva
	// (requiere sesión: el préstamo se registra a nombre de quien se autentica)
	mux.HandleFunc("POST /api/prestamos", s.conAuth(s.crearPrestamo))

	// Endpoint 7: POST /api/prestamos/{id}/devolver - devolver un libro
	// (requiere sesión: solo el dueño del préstamo o un administrador)
	mux.HandleFunc("POST /api/prestamos/{id}/devolver", s.conAuth(s.devolverPrestamo))

	// Endpoint 8: GET /api/reportes/mas-prestados - reporte top N
	mux.HandleFunc("GET /api/reportes/mas-prestados", s.reporteMasPrestados)

	// Endpoint bonus: GET /api/reservas/{libro_id} - ver cola de un libro
	mux.HandleFunc("GET /api/reservas/{libro_id}", s.consultarCola)

	// Endpoint bonus: GET /api/categorias - listar categorías
	mux.HandleFunc("GET /api/categorias", s.listarCategorias)

	// Endpoint raíz con información del API
	mux.HandleFunc("GET /", s.raiz)

	return mux
}

// ---------------------------------------------------------------------
// DTOs: estructuras solo para serialización JSON. Al ser públicas
// permiten que encoding/json las serialice sin exponer los tipos
// internos del dominio.
// ---------------------------------------------------------------------

type libroDTO struct {
	ID          int    `json:"id"`
	Titulo      string `json:"titulo"`
	Autor       string `json:"autor"`
	CategoriaID int    `json:"categoria_id"`
	Anio        int    `json:"anio"`
	Formato     string `json:"formato"`
	Disponible  bool   `json:"disponible"`
}

func libroADTO(l *catalogo.Libro) libroDTO {
	return libroDTO{
		ID:          l.ID(),
		Titulo:      l.Titulo(),
		Autor:       l.Autor(),
		CategoriaID: l.CategoriaID(),
		Anio:        l.Anio(),
		Formato:     string(l.Formato()),
		Disponible:  l.Disponible(),
	}
}

type prestamoDTO struct {
	ID               int    `json:"id"`
	UsuarioID        int    `json:"usuario_id"`
	LibroID          int    `json:"libro_id"`
	FechaInicio      string `json:"fecha_inicio"`
	FechaVencimiento string `json:"fecha_vencimiento"`
	Estado           string `json:"estado"`
}

func prestamoADTO(p *prestamos.Prestamo) prestamoDTO {
	return prestamoDTO{
		ID:               p.ID(),
		UsuarioID:        p.UsuarioID(),
		LibroID:          p.LibroID(),
		FechaInicio:      p.FechaInicio().Format("2006-01-02"),
		FechaVencimiento: p.FechaVencimiento().Format("2006-01-02"),
		Estado:           string(p.Estado()),
	}
}

type recomendacionDTO struct {
	Libro libroDTO `json:"libro"`
	Score float64  `json:"score"`
	Razon string   `json:"razon"`
}

type errorDTO struct {
	Error string `json:"error"`
}

// ---------------------------------------------------------------------
// Helpers de respuesta
// ---------------------------------------------------------------------

// enviarJSON serializa el valor y lo envía como respuesta JSON con el
// código de estado indicado. Centraliza el manejo de errores de
// serialización y los headers.
func enviarJSON(w http.ResponseWriter, codigo int, valor interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	if err := json.NewEncoder(w).Encode(valor); err != nil {
		http.Error(w, "error al serializar respuesta", http.StatusInternalServerError)
	}
}

// enviarError responde con un JSON de error y el código HTTP correspondiente.
func enviarError(w http.ResponseWriter, codigo int, mensaje string) {
	enviarJSON(w, codigo, errorDTO{Error: mensaje})
}

// ---------------------------------------------------------------------
// Middleware de autenticación
// ---------------------------------------------------------------------

// handlerAutenticado es como un http.HandlerFunc pero recibe además el
// usuario que ya fue identificado. Gracias a este tipo, los handlers
// protegidos no repiten la comprobación de credenciales: cuando se ejecutan,
// el usuario ya es de fiar.
type handlerAutenticado func(http.ResponseWriter, *http.Request, *usuarios.Usuario)

// conAuth envuelve un handler para exigir credenciales válidas. Es el patrón
// middleware: una función que recibe un handler y devuelve otro handler con
// comportamiento añadido alrededor del original.
//
// Se usa autenticación básica de HTTP (r.BasicAuth) porque viene en la
// biblioteca estándar y no obliga a inventar un formato propio de token.
func (s *Server) conAuth(siguiente handlerAutenticado) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correo, password, ok := r.BasicAuth()
		if !ok {
			// El header se escribe antes que el cuerpo: una vez enviado el
			// código de estado ya no se pueden añadir cabeceras.
			w.Header().Set("WWW-Authenticate", `Basic realm="LeeLibre"`)
			enviarError(w, http.StatusUnauthorized, "se requieren credenciales")
			return
		}
		u, err := s.autenticador.Autenticar(correo, password)
		if err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="LeeLibre"`)
			// Se reenvía el error del autenticador tal cual, que no distingue
			// entre correo inexistente y contraseña incorrecta a propósito.
			enviarError(w, http.StatusUnauthorized, err.Error())
			return
		}
		siguiente(w, r, u)
	}
}

// conAdmin exige, además de credenciales válidas, el rol de administrador.
// Se construye reutilizando conAuth en lugar de repetir su lógica: primero
// se resuelve quién es (401 si no se sabe) y después si puede (403 si no).
func (s *Server) conAdmin(siguiente handlerAutenticado) http.HandlerFunc {
	return s.conAuth(func(w http.ResponseWriter, r *http.Request, u *usuarios.Usuario) {
		if !u.EsAdministrador() {
			enviarError(w, http.StatusForbidden, "se requiere rol de administrador")
			return
		}
		siguiente(w, r, u)
	})
}

// ---------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------

// raiz responde con información básica del API.
//
// Esta lista se escribe a mano porque http.ServeMux no permite consultar las
// rutas que tiene registradas. Es decir, está duplicada respecto a Rutas(): al
// agregar o quitar un endpoint allí, hay que actualizarla también aquí o el
// API acabará anunciando algo distinto de lo que ofrece.
func (s *Server) raiz(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"servicio":      "LeeLibre API",
		"version":       "1.1.0",
		"autenticacion": "HTTP Basic en los endpoints marcados; el resto son públicos",
		"endpoints": []string{
			"GET    /",
			"GET    /api/libros",
			"GET    /api/libros/{id}",
			"POST   /api/libros                        (administrador)",
			"PUT    /api/libros/{id}                   (administrador)",
			"DELETE /api/libros/{id}                   (administrador)",
			"GET    /api/categorias",
			"POST   /api/prestamos                     (autenticado)",
			"POST   /api/prestamos/{id}/devolver       (dueño o administrador)",
			"GET    /api/usuarios/{id}/recomendaciones",
			"GET    /api/reportes/mas-prestados",
			"GET    /api/reservas/{libro_id}",
		},
	}
	enviarJSON(w, http.StatusOK, info)
}

// listarLibros - GET /api/libros
// Soporta filtro opcional por query string: ?texto=X o ?categoria=Y
func (s *Server) listarLibros(w http.ResponseWriter, r *http.Request) {
	texto := r.URL.Query().Get("texto")
	catStr := r.URL.Query().Get("categoria")

	var libros []*catalogo.Libro
	if texto != "" {
		libros = s.catalogo.BuscarPorTexto(texto)
	} else if catStr != "" {
		catID, err := strconv.Atoi(catStr)
		if err != nil {
			enviarError(w, http.StatusBadRequest, "categoria debe ser un número")
			return
		}
		libros = s.catalogo.PorCategoria(catID)
	} else {
		libros = s.catalogo.Todos()
	}

	dtos := make([]libroDTO, 0, len(libros))
	for _, l := range libros {
		dtos = append(dtos, libroADTO(l))
	}
	enviarJSON(w, http.StatusOK, dtos)
}

// obtenerLibro - GET /api/libros/{id}
func (s *Server) obtenerLibro(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		enviarError(w, http.StatusBadRequest, "id debe ser un número")
		return
	}
	libro, err := s.catalogo.BuscarPorID(id)
	if err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}
	enviarJSON(w, http.StatusOK, libroADTO(libro))
}

// crearLibro - POST /api/libros
// El tercer parámetro es el administrador que realiza la operación; no se
// usa aquí, pero la firma es la que exige conAdmin.
func (s *Server) crearLibro(w http.ResponseWriter, r *http.Request, _ *usuarios.Usuario) {
	var entrada libroDTO
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		enviarError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	libro, err := catalogo.NewLibro(
		entrada.ID, entrada.Titulo, entrada.Autor,
		entrada.CategoriaID, entrada.Anio, catalogo.Formato(entrada.Formato),
	)
	if err != nil {
		enviarError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.catalogo.AgregarLibro(libro); err != nil {
		enviarError(w, http.StatusBadRequest, err.Error())
		return
	}
	enviarJSON(w, http.StatusCreated, libroADTO(libro))
}

// actualizarLibro - PUT /api/libros/{id}
// Body: cualquier subconjunto de {"titulo", "autor", "anio", "formato"}.
//
// Los campos se reciben como punteros para distinguir "no lo envíes" de
// "ponlo vacío": si llegaran como valores normales, un título ausente sería
// indistinguible de un título en blanco y se sobrescribiría sin querer.
//
// La validación no se repite aquí: cada setter del dominio aplica su propia
// regla y devuelve el error, de modo que la capa web solo lo traduce a un
// código HTTP.
func (s *Server) actualizarLibro(w http.ResponseWriter, r *http.Request, _ *usuarios.Usuario) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		enviarError(w, http.StatusBadRequest, "id debe ser un número")
		return
	}
	libro, err := s.catalogo.BuscarPorID(id)
	if err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}
	var entrada struct {
		Titulo  *string `json:"titulo"`
		Autor   *string `json:"autor"`
		Anio    *int    `json:"anio"`
		Formato *string `json:"formato"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		enviarError(w, http.StatusBadRequest, "JSON inválido: "+err.Error())
		return
	}
	if entrada.Titulo != nil {
		if err := libro.SetTitulo(*entrada.Titulo); err != nil {
			enviarError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if entrada.Autor != nil {
		if err := libro.SetAutor(*entrada.Autor); err != nil {
			enviarError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if entrada.Anio != nil {
		if err := libro.SetAnio(*entrada.Anio); err != nil {
			enviarError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if entrada.Formato != nil {
		if err := libro.SetFormato(catalogo.Formato(*entrada.Formato)); err != nil {
			enviarError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	enviarJSON(w, http.StatusOK, libroADTO(libro))
}

// eliminarLibro - DELETE /api/libros/{id}
func (s *Server) eliminarLibro(w http.ResponseWriter, r *http.Request, _ *usuarios.Usuario) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		enviarError(w, http.StatusBadRequest, "id debe ser un número")
		return
	}
	if err := s.catalogo.EliminarLibro(id); err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// recomendaciones - GET /api/usuarios/{id}/recomendaciones
// Esta es la funcionalidad "no básica": recibe un usuario y calcula
// recomendaciones personalizadas basadas en su historial de préstamos.
func (s *Server) recomendaciones(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		enviarError(w, http.StatusBadRequest, "id debe ser un número")
		return
	}
	usuario, err := s.autenticador.BuscarPorID(id)
	if err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}
	rec := reportes.NewRecomendador(s.catalogo, s.historial, usuario, 5)
	recomendaciones, err := rec.Recomendar()
	if err != nil {
		enviarError(w, http.StatusOK, err.Error())
		return
	}
	dtos := make([]recomendacionDTO, 0, len(recomendaciones))
	for _, r := range recomendaciones {
		dtos = append(dtos, recomendacionDTO{
			Libro: libroADTO(r.Libro),
			Score: r.Score,
			Razon: r.Razon,
		})
	}
	enviarJSON(w, http.StatusOK, dtos)
}

// crearPrestamo - POST /api/prestamos
// Body: {"libro_id": M}
// Si el libro está disponible: crea un préstamo.
// Si no está disponible: agrega al usuario a la cola de reservas.
//
// El préstamo se registra siempre a nombre de quien se autentica, no de un
// usuario_id enviado en el cuerpo: si el cliente pudiera elegirlo, cualquiera
// podría pedir libros a nombre de otra persona. Por eso ya no se valida que
// el usuario exista, porque autenticarse lo demuestra.
func (s *Server) crearPrestamo(w http.ResponseWriter, r *http.Request, actor *usuarios.Usuario) {
	var entrada struct {
		LibroID int `json:"libro_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&entrada); err != nil {
		enviarError(w, http.StatusBadRequest, "JSON inválido")
		return
	}
	libro, err := s.catalogo.BuscarPorID(entrada.LibroID)
	if err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}

	if !libro.Disponible() {
		// Rama del diagrama: no disponible -> cola de reservas
		if err := s.gestorReservas.Reservar(entrada.LibroID, actor.ID()); err != nil {
			enviarError(w, http.StatusConflict, err.Error())
			return
		}
		pos := s.gestorReservas.PosicionEnCola(entrada.LibroID, actor.ID())
		enviarJSON(w, http.StatusAccepted, map[string]interface{}{
			"mensaje":  "libro no disponible, agregado a la cola de reservas",
			"posicion": pos,
			"libro_id": entrada.LibroID,
		})
		return
	}

	if err := libro.Prestar(); err != nil {
		enviarError(w, http.StatusConflict, err.Error())
		return
	}
	p, err := s.historial.Registrar(actor.ID(), entrada.LibroID)
	if err != nil {
		libro.Devolver()
		enviarError(w, http.StatusInternalServerError, err.Error())
		return
	}
	enviarJSON(w, http.StatusCreated, prestamoADTO(p))
}

// devolverPrestamo - POST /api/prestamos/{id}/devolver
func (s *Server) devolverPrestamo(w http.ResponseWriter, r *http.Request, actor *usuarios.Usuario) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		enviarError(w, http.StatusBadRequest, "id debe ser un número")
		return
	}
	p, err := s.historial.BuscarPorID(id)
	if err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}
	// Autenticarse no basta: hay que ser el dueño del préstamo. De lo
	// contrario, cualquier lector podría devolver los libros de los demás.
	// El administrador queda exento porque gestiona el catálogo completo.
	if p.UsuarioID() != actor.ID() && !actor.EsAdministrador() {
		enviarError(w, http.StatusForbidden, "el préstamo pertenece a otro usuario")
		return
	}
	if err := p.Devolver(); err != nil {
		enviarError(w, http.StatusConflict, err.Error())
		return
	}
	respuesta := map[string]interface{}{
		"mensaje":  "libro devuelto",
		"prestamo": prestamoADTO(p),
	}
	if libro, err := s.catalogo.BuscarPorID(p.LibroID()); err == nil {
		libro.Devolver()
		// Si había cola, notificar al siguiente
		siguienteID, err := s.gestorReservas.LiberarLibro(p.LibroID())
		if err == nil && siguienteID > 0 {
			respuesta["siguiente_en_cola"] = siguienteID
		}
	}
	enviarJSON(w, http.StatusOK, respuesta)
}

// reporteMasPrestados - GET /api/reportes/mas-prestados
func (s *Server) reporteMasPrestados(w http.ResponseWriter, r *http.Request) {
	limiteStr := r.URL.Query().Get("limite")
	limite := 5
	if limiteStr != "" {
		if n, err := strconv.Atoi(limiteStr); err == nil && n > 0 {
			limite = n
		}
	}
	// Uso directo del reporteador; el texto formateado se envía como campo.
	// Polimorfismo en acción: r puede ser cualquier Reportador.
	var rep reportes.Reportador = reportes.NewMasPrestados(s.catalogo, s.historial, limite)
	enviarJSON(w, http.StatusOK, map[string]interface{}{
		"reporte":   rep.Nombre(),
		"contenido": rep.Generar(),
	})
}

// consultarCola - GET /api/reservas/{libro_id}
func (s *Server) consultarCola(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("libro_id"))
	if err != nil {
		enviarError(w, http.StatusBadRequest, "libro_id debe ser un número")
		return
	}
	libro, err := s.catalogo.BuscarPorID(id)
	if err != nil {
		enviarError(w, http.StatusNotFound, err.Error())
		return
	}
	cola := s.gestorReservas.ColaDe(id)
	enviarJSON(w, http.StatusOK, map[string]interface{}{
		"libro":       libroADTO(libro),
		"tamano_cola": cola.Tamano(),
		"usuarios":    cola.Elementos(),
	})
}

// listarCategorias - GET /api/categorias
func (s *Server) listarCategorias(w http.ResponseWriter, r *http.Request) {
	cats := s.catalogo.TodasCategorias()
	tipo := []map[string]interface{}{}
	for _, c := range cats {
		tipo = append(tipo, map[string]interface{}{
			"id":     c.ID(),
			"nombre": c.Nombre(),
		})
	}
	enviarJSON(w, http.StatusOK, tipo)
}

// ---------------------------------------------------------------------
// Verificación defensiva: si el paquete errores no se usa, el import
// silencia el warning. Se usa en el helper de abajo (más adelante para
// mapear errores de dominio a códigos HTTP semánticos).
// ---------------------------------------------------------------------

// codigoHTTPDe mapea un error de dominio al código HTTP apropiado.
// Se declara como función interna del paquete pero no siempre se usa
// externamente; queda como utilidad para futura expansión.
func codigoHTTPDe(err error) int {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "no encontrado"), strings.Contains(msg, "no existe"):
		return http.StatusNotFound
	case strings.Contains(msg, "no disponible"), strings.Contains(msg, "ya"):
		return http.StatusConflict
	case strings.Contains(msg, "inválido"):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// Uso silencioso para evitar el warning "imported and not used"
var _ = errores.ErrDatosInvalidos
var _ = fmt.Sprintf
var _ = codigoHTTPDe
