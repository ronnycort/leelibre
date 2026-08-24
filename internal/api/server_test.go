package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ronnycort/leelibre/internal/catalogo"
	"github.com/ronnycort/leelibre/internal/prestamos"
	"github.com/ronnycort/leelibre/internal/reservas"
	"github.com/ronnycort/leelibre/internal/usuarios"
)

// servidorDePrueba arma un servidor completo con datos conocidos, sin leer
// ningún archivo. Las pruebas no dependen así del contenido de data/ ni
// dejan rastro en el disco.
func servidorDePrueba(t *testing.T) http.Handler {
	t.Helper()

	cat := catalogo.NewCatalogo()
	ficcion, _ := catalogo.NewCategoria(1, "Ficción")
	cat.AgregarCategoria(ficcion)
	for i := 1; i <= 3; i++ {
		l, err := catalogo.NewLibro(i, "Libro "+string(rune('A'+i-1)), "Autora", 1, 2000, catalogo.FormatoPDF)
		if err != nil {
			t.Fatalf("preparación fallida: %v", err)
		}
		cat.AgregarLibro(l)
	}

	auth := usuarios.NewAutenticador()
	lector, _ := usuarios.NewUsuario(1, "Ana", "ana@test.ec", "ana123", usuarios.RolLector)
	otro, _ := usuarios.NewUsuario(2, "Carlos", "carlos@test.ec", "carlos123", usuarios.RolLector)
	admin, _ := usuarios.NewUsuario(3, "Ronny", "ronny@test.ec", "ronny123", usuarios.RolAdministrador)
	auth.Registrar(lector)
	auth.Registrar(otro)
	auth.Registrar(admin)

	s := NewServer(cat, auth, prestamos.NewHistorial(), reservas.NewGestorReservas())
	return s.Rutas()
}

// hacer envía una petición al servidor sin abrir ningún puerto de red.
// httptest.NewRecorder implementa http.ResponseWriter en memoria, así que el
// handler escribe en él como si fuera una respuesta real y después se puede
// inspeccionar el código y el cuerpo. Es lo que hace estas pruebas instantáneas.
func hacer(t *testing.T, h http.Handler, metodo, ruta, cuerpo, usuario, clave string) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if cuerpo == "" {
		req = httptest.NewRequest(metodo, ruta, nil)
	} else {
		req = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
		req.Header.Set("Content-Type", "application/json")
	}
	if usuario != "" {
		req.SetBasicAuth(usuario, clave)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestEndpointsPublicosRespondenOK(t *testing.T) {
	h := servidorDePrueba(t)

	rutas := []string{
		"/",
		"/api/libros",
		"/api/libros/1",
		"/api/categorias",
		"/api/reportes/mas-prestados",
		"/api/reservas/1",
	}

	for _, ruta := range rutas {
		t.Run(ruta, func(t *testing.T) {
			rec := hacer(t, h, http.MethodGet, ruta, "", "", "")
			if rec.Code != http.StatusOK {
				t.Errorf("se esperaba 200, se obtuvo %d", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Errorf("la respuesta debería ser JSON, su Content-Type es %q", ct)
			}
		})
	}
}

// TestListarLibrosDevuelveJSONValido comprueba la serialización: no basta con
// que responda 200, el cuerpo tiene que ser JSON que se pueda deserializar.
func TestListarLibrosDevuelveJSONValido(t *testing.T) {
	h := servidorDePrueba(t)
	rec := hacer(t, h, http.MethodGet, "/api/libros", "", "", "")

	var libros []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &libros); err != nil {
		t.Fatalf("la respuesta no es JSON válido: %v", err)
	}
	if len(libros) != 3 {
		t.Errorf("se esperaban 3 libros, llegaron %d", len(libros))
	}
	if _, existe := libros[0]["titulo"]; !existe {
		t.Error("al JSON del libro le falta el campo 'titulo'")
	}
}

func TestLibroInexistenteDevuelve404(t *testing.T) {
	h := servidorDePrueba(t)
	rec := hacer(t, h, http.MethodGet, "/api/libros/999", "", "", "")

	if rec.Code != http.StatusNotFound {
		t.Errorf("se esperaba 404, se obtuvo %d", rec.Code)
	}
}

func TestIdNoNumericoDevuelve400(t *testing.T) {
	h := servidorDePrueba(t)
	rec := hacer(t, h, http.MethodGet, "/api/libros/abc", "", "", "")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("se esperaba 400, se obtuvo %d", rec.Code)
	}
}

// TestAutorizacion recorre la matriz completa de permisos. Es la prueba que
// distingue las tres preguntas: quién eres (401), si puedes (403) y si puedes
// sobre ese recurso concreto.
func TestAutorizacion(t *testing.T) {
	casos := []struct {
		nombre   string
		metodo   string
		ruta     string
		cuerpo   string
		usuario  string
		clave    string
		esperado int
	}{
		{"borrar sin credenciales", http.MethodDelete, "/api/libros/3", "", "", "", http.StatusUnauthorized},
		{"borrar con contraseña incorrecta", http.MethodDelete, "/api/libros/3", "", "ronny@test.ec", "mala", http.StatusUnauthorized},
		{"borrar siendo lector", http.MethodDelete, "/api/libros/3", "", "ana@test.ec", "ana123", http.StatusForbidden},
		{"borrar siendo administrador", http.MethodDelete, "/api/libros/3", "", "ronny@test.ec", "ronny123", http.StatusNoContent},
		{"prestar sin credenciales", http.MethodPost, "/api/prestamos", `{"libro_id":1}`, "", "", http.StatusUnauthorized},
		{"prestar autenticado", http.MethodPost, "/api/prestamos", `{"libro_id":1}`, "ana@test.ec", "ana123", http.StatusCreated},
		{"modificar siendo lector", http.MethodPut, "/api/libros/2", `{"anio":1990}`, "ana@test.ec", "ana123", http.StatusForbidden},
		{"modificar siendo administrador", http.MethodPut, "/api/libros/2", `{"anio":1990}`, "ronny@test.ec", "ronny123", http.StatusOK},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			// Servidor nuevo por caso: si compartieran uno, borrar el libro 3
			// en un caso afectaría a los siguientes.
			h := servidorDePrueba(t)
			rec := hacer(t, h, c.metodo, c.ruta, c.cuerpo, c.usuario, c.clave)
			if rec.Code != c.esperado {
				t.Errorf("se esperaba %d, se obtuvo %d — cuerpo: %s", c.esperado, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestNoSePuedeSuplantarAOtroUsuario es la prueba de seguridad más importante
// del API: aunque el cliente envíe un usuario_id ajeno en el cuerpo, el
// préstamo debe quedar a nombre de quien se autenticó.
func TestNoSePuedeSuplantarAOtroUsuario(t *testing.T) {
	h := servidorDePrueba(t)

	// Ana (id 1) intenta pedir el libro a nombre del usuario 3.
	rec := hacer(t, h, http.MethodPost, "/api/prestamos", `{"usuario_id":3,"libro_id":1}`, "ana@test.ec", "ana123")
	if rec.Code != http.StatusCreated {
		t.Fatalf("se esperaba 201, se obtuvo %d", rec.Code)
	}

	var prestamo map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &prestamo); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if id, _ := prestamo["usuario_id"].(float64); int(id) != 1 {
		t.Errorf("el préstamo quedó a nombre del usuario %v; debía ser el 1 (Ana), que es quien se autenticó", prestamo["usuario_id"])
	}
}

// TestDevolverPrestamoAjeno comprueba la autorización a nivel de recurso.
func TestDevolverPrestamoAjeno(t *testing.T) {
	h := servidorDePrueba(t)

	rec := hacer(t, h, http.MethodPost, "/api/prestamos", `{"libro_id":1}`, "ana@test.ec", "ana123")
	if rec.Code != http.StatusCreated {
		t.Fatalf("preparación fallida: %d", rec.Code)
	}
	var prestamo map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &prestamo)
	id := int(prestamo["id"].(float64))
	ruta := "/api/prestamos/" + itoa(id) + "/devolver"

	// Carlos intenta devolver el préstamo de Ana.
	rec = hacer(t, h, http.MethodPost, ruta, "", "carlos@test.ec", "carlos123")
	if rec.Code != http.StatusForbidden {
		t.Errorf("un usuario pudo devolver el préstamo de otro: código %d", rec.Code)
	}

	// Ana sí puede devolver el suyo.
	rec = hacer(t, h, http.MethodPost, ruta, "", "ana@test.ec", "ana123")
	if rec.Code != http.StatusOK {
		t.Errorf("la dueña no pudo devolver su propio préstamo: código %d", rec.Code)
	}
}

// TestLibroNoDisponibleEncolaAlUsuario comprueba la rama del diagrama de
// flujo: si el libro está prestado, el segundo usuario entra en la cola.
func TestLibroNoDisponibleEncolaAlUsuario(t *testing.T) {
	h := servidorDePrueba(t)

	if rec := hacer(t, h, http.MethodPost, "/api/prestamos", `{"libro_id":1}`, "ana@test.ec", "ana123"); rec.Code != http.StatusCreated {
		t.Fatalf("preparación fallida: %d", rec.Code)
	}

	rec := hacer(t, h, http.MethodPost, "/api/prestamos", `{"libro_id":1}`, "carlos@test.ec", "carlos123")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("se esperaba 202 (encolado), se obtuvo %d — cuerpo: %s", rec.Code, rec.Body.String())
	}

	var respuesta map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &respuesta)
	if pos, _ := respuesta["posicion"].(float64); int(pos) != 1 {
		t.Errorf("Carlos debería quedar en la posición 1, quedó en %v", respuesta["posicion"])
	}
}

// TestResumenConcurrenteDevuelveTodosLosReportes comprueba el endpoint que
// genera reportes en paralelo: deben llegar todos, sin perderse ninguno.
func TestResumenConcurrenteDevuelveTodosLosReportes(t *testing.T) {
	h := servidorDePrueba(t)
	rec := hacer(t, h, http.MethodGet, "/api/reportes/resumen?usuario=1", "", "", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("se esperaba 200, se obtuvo %d", rec.Code)
	}
	var respuesta map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &respuesta); err != nil {
		t.Fatalf("la respuesta no es JSON válido: %v", err)
	}
	if n, _ := respuesta["generados"].(float64); int(n) != 2 {
		t.Errorf("se esperaban 2 reportes generados, llegaron %v", respuesta["generados"])
	}
}

func TestJSONMalFormadoDevuelve400(t *testing.T) {
	h := servidorDePrueba(t)
	rec := hacer(t, h, http.MethodPost, "/api/prestamos", `{esto no es json`, "ana@test.ec", "ana123")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("se esperaba 400 ante un JSON inválido, se obtuvo %d", rec.Code)
	}
}

// itoa evita importar strconv solo para esto en el archivo de pruebas.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digitos := ""
	for n > 0 {
		digitos = string(rune('0'+n%10)) + digitos
		n /= 10
	}
	return digitos
}
