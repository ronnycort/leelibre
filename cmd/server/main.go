// Programa servidor de LeeLibre: expone las funcionalidades del sistema
// mediante servicios web REST con serialización JSON.
//
// Comparte los mismos paquetes del dominio con la CLI (cmd/leelibre),
// demostrando que la arquitectura por capas permite exponer la misma
// lógica a través de múltiples interfaces sin duplicación.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ronnycort/leelibre/internal/api"
	"github.com/ronnycort/leelibre/internal/catalogo"
	"github.com/ronnycort/leelibre/internal/prestamos"
	"github.com/ronnycort/leelibre/internal/reservas"
	"github.com/ronnycort/leelibre/internal/usuarios"
)

const (
	rutaCategorias = "data/categorias.json"
	rutaLibros     = "data/libros.json"
	rutaUsuarios   = "data/usuarios.json"
)

func main() {
	// Determinar puerto (por defecto 8080, se puede sobrescribir con env var)
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8080"
	}

	// Carga concurrente de los datos iniciales.
	//
	// El catálogo y los usuarios viven en archivos distintos y no dependen
	// entre sí, así que se leen en paralelo en lugar de uno después del otro.
	// Cada goroutine envía su error al canal —o nil si todo fue bien— y el
	// programa continúa cuando ambas han terminado.
	//
	// Las categorías y los libros sí se cargan en secuencia dentro de la
	// misma goroutine, porque un libro no puede registrarse si su categoría
	// todavía no existe. Es un ejemplo de que no todo se puede paralelizar:
	// donde hay dependencia, el orden importa.
	cat := catalogo.NewCatalogo()
	auth := usuarios.NewAutenticador()

	var wg sync.WaitGroup
	errores := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		errores <- cat.CargarDesdeArchivos(rutaCategorias, rutaLibros)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		errores <- auth.CargarDesdeArchivo(rutaUsuarios)
	}()

	wg.Wait()
	close(errores)

	for err := range errores {
		if err != nil {
			log.Fatalf("Error cargando los datos iniciales: %v", err)
		}
	}

	hist := prestamos.NewHistorial()
	sembrarHistorial(hist, cat)
	hist.ActualizarEstados()

	gr := reservas.NewGestorReservas()

	// Construir servidor y registrar rutas
	servidor := api.NewServer(cat, auth, hist, gr)
	mux := servidor.Rutas()

	// Middleware simple para registrar cada request en consola
	loggedMux := loggingMiddleware(mux)

	direccion := ":" + puerto
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Println("║       LeeLibre API - Servicios REST        ║")
	fmt.Println("╚════════════════════════════════════════════╝")
	fmt.Printf("\n  Servidor escuchando en http://localhost%s\n", direccion)
	fmt.Println("\n  Endpoints disponibles:")
	fmt.Println("    GET    /                                  - información del API")
	fmt.Println("    GET    /api/libros                        - listar catálogo")
	fmt.Println("    GET    /api/libros/{id}                   - detalle de un libro")
	fmt.Println("    POST   /api/libros                        - crear libro")
	fmt.Println("    DELETE /api/libros/{id}                   - eliminar libro")
	fmt.Println("    GET    /api/categorias                    - listar categorías")
	fmt.Println("    POST   /api/prestamos                     - registrar préstamo")
	fmt.Println("    POST   /api/prestamos/{id}/devolver       - devolver libro")
	fmt.Println("    GET    /api/usuarios/{id}/recomendaciones - recomendaciones")
	fmt.Println("    GET    /api/reportes/mas-prestados        - top de libros")
	fmt.Println("    GET    /api/reservas/{libro_id}           - cola de reservas")
	fmt.Println("\n  Presiona Ctrl+C para detener el servidor.")
	fmt.Println()

	if err := http.ListenAndServe(direccion, loggedMux); err != nil {
		log.Fatalf("Error del servidor: %v", err)
	}
}

// sembrarHistorial es igual que en la CLI - carga préstamos históricos
// para que el recomendador tenga datos con que trabajar.
func sembrarHistorial(hist *prestamos.Historial, cat *catalogo.Catalogo) {
	prestamosSemilla := []struct {
		usuario int
		libro   int
		veces   int
	}{
		{2, 11, 3}, {3, 11, 2}, {4, 11, 1},
		{2, 1, 2}, {5, 1, 1}, {6, 1, 1},
		{2, 2, 2}, {7, 2, 1},
		{2, 3, 2},
		{3, 6, 2}, {3, 7, 1}, {3, 8, 1},
		{4, 15, 1}, {4, 16, 1},
		{5, 19, 1}, {5, 20, 1},
	}
	for _, ps := range prestamosSemilla {
		for i := 0; i < ps.veces; i++ {
			p, err := hist.Registrar(ps.usuario, ps.libro)
			if err != nil {
				continue
			}
			_ = p.Devolver()
			if libro, err := cat.BuscarPorID(ps.libro); err == nil {
				libro.Devolver()
			}
		}
	}
}

// loggingMiddleware es un middleware simple que registra cada request
// entrante con método, ruta y tiempo de respuesta.
// Es un ejemplo idiomático de middleware en net/http: una función que
// envuelve un http.Handler y retorna otro http.Handler.
// peticionesAtendidas cuenta cuántas peticiones ha servido el servidor desde
// que arrancó. Es un contador compartido por todas las goroutines que atienden
// peticiones, así que no puede ser un int normal: dos peticiones simultáneas
// harían "leer, sumar uno, escribir" a la vez y una de las dos sumas se
// perdería.
//
// atomic.Int64 resuelve justamente eso. Su método Add es una operación
// atómica, es decir, indivisible: ninguna otra goroutine puede colarse en
// medio. Aquí basta con esto y no hace falta un mutex, porque la única
// operación sobre el contador es sumar.
var peticionesAtendidas atomic.Int64

func loggingMiddleware(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		n := peticionesAtendidas.Add(1)
		siguiente.ServeHTTP(w, r)
		log.Printf("#%d %s %s -- %v", n, r.Method, r.URL.Path, time.Since(inicio))
	})
}
