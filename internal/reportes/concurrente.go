package reportes

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ResultadoReporte es lo que produce un reporte al ejecutarse. Viaja por un
// canal desde la goroutine que lo generó hasta la que recoge los resultados.
type ResultadoReporte struct {
	Orden     int           // posición original, para no perder el orden al recogerlos
	Nombre    string        // título del reporte
	Contenido string        // texto ya formateado
	Duracion  time.Duration // cuánto tardó en generarse
}

// EjecutarTodosConcurrente genera varios reportes al mismo tiempo, uno por
// goroutine, en lugar de esperar a que termine cada uno para empezar el
// siguiente como hace EjecutarTodos.
//
// Tiene sentido aquí porque los reportes son independientes entre sí: el
// recomendador no necesita el resultado del top de préstamos ni al revés.
// Cuando un reporte tarda —el recomendador recorre el historial completo de
// un usuario y después todo el catálogo— el resto no se queda esperando.
//
// El patrón es el de la clase:
//
//   - un sync.WaitGroup para saber cuándo terminaron todas las goroutines,
//   - un canal por donde cada una entrega su resultado,
//   - defer wg.Done() como primera línea de la goroutine, para que se marque
//     como terminada aunque salga por un camino distinto al esperado.
func EjecutarTodosConcurrente(reportes []Reportador) []ResultadoReporte {
	var wg sync.WaitGroup

	// El canal se crea con capacidad para todos los resultados (buffered).
	// Sin buffer, cada goroutine quedaría bloqueada al escribir hasta que
	// alguien leyera, y como aquí se lee después del Wait, se bloquearían
	// todas y el programa no avanzaría.
	canal := make(chan ResultadoReporte, len(reportes))

	for i, r := range reportes {
		wg.Add(1) // se registra ANTES de lanzar la goroutine, no dentro

		// i y r se pasan como parámetros en lugar de capturarse del bucle:
		// así cada goroutine recibe su propia copia y no comparte la
		// variable del for con las demás.
		go func(orden int, rep Reportador) {
			defer wg.Done()

			inicio := time.Now()
			contenido := rep.Generar()

			canal <- ResultadoReporte{
				Orden:     orden,
				Nombre:    rep.Nombre(),
				Contenido: contenido,
				Duracion:  time.Since(inicio),
			}
		}(i, r)
	}

	wg.Wait()    // espera a que las goroutines terminen
	close(canal) // cerrar permite recorrer el canal con range sin bloquearse

	resultados := make([]ResultadoReporte, 0, len(reportes))
	for res := range canal {
		resultados = append(resultados, res)
	}

	// Las goroutines terminan en cualquier orden, así que el resultado se
	// reordena por el índice original para que la salida sea siempre la
	// misma. Sin esto, dos ejecuciones seguidas darían reportes en distinto
	// orden y el programa dejaría de ser predecible.
	sort.Slice(resultados, func(a, b int) bool {
		return resultados[a].Orden < resultados[b].Orden
	})
	return resultados
}

// FormatearResultados arma el texto final a partir de los reportes ya
// generados, para la salida por consola.
func FormatearResultados(resultados []ResultadoReporte) string {
	texto := ""
	for _, r := range resultados {
		texto += fmt.Sprintf("\n=== %s (generado en %v) ===\n%s\n", r.Nombre, r.Duracion, r.Contenido)
	}
	return texto
}
