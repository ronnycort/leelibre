package reportes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ronnycort/leelibre/internal/catalogo"
	"github.com/ronnycort/leelibre/internal/prestamos"
)

// MasPrestados es un reporte que muestra los libros más prestados del
// sistema, ordenados por número de préstamos descendente.
// Implementa la interface Reportador.
type MasPrestados struct {
	catalogo  *catalogo.Catalogo
	historial *prestamos.Historial
	limite    int
}

// NewMasPrestados construye el reporte. El limite indica cuántos libros
// mostrar (top N).
func NewMasPrestados(cat *catalogo.Catalogo, hist *prestamos.Historial, limite int) *MasPrestados {
	if limite <= 0 {
		limite = 5
	}
	return &MasPrestados{
		catalogo:  cat,
		historial: hist,
		limite:    limite,
	}
}

// Nombre implementa parte de la interface Reportador.
func (r *MasPrestados) Nombre() string {
	return fmt.Sprintf("Top %d libros más prestados", r.limite)
}

// Generar recorre los préstamos, cuenta ocurrencias por libro y arma
// una lista ordenada por conteo. Aquí se ve el uso combinado de:
// - map[int]int (semana 3)
// - slice + sort.Slice (funciones como valores)
// - strings.Builder (eficiencia en concatenación)
func (r *MasPrestados) Generar() string {
	conteo := r.historial.ConteoPorLibro()

	// Convertir map a slice para poder ordenar
	type entrada struct {
		libroID int
		veces   int
	}
	entradas := make([]entrada, 0, len(conteo))
	for id, veces := range conteo {
		entradas = append(entradas, entrada{id, veces})
	}

	// sort.Slice recibe una función de comparación — de nuevo, funciones
	// como argumentos son un patrón central en Go, sin que eso lo haga
	// un lenguaje funcional. Aquí es simplemente una convención.
	sort.Slice(entradas, func(i, j int) bool {
		return entradas[i].veces > entradas[j].veces
	})

	var sb strings.Builder
	if len(entradas) == 0 {
		return "No hay préstamos registrados todavía.\n"
	}
	// Recorremos hasta encontrar `limite` libros existentes, no las
	// primeras `limite` entradas (algunas pueden apuntar a libros ya
	// eliminados y en ese caso se saltan sin gastar cupo).
	posicion := 0
	for _, e := range entradas {
		if posicion >= r.limite {
			break
		}
		libro, err := r.catalogo.BuscarPorID(e.libroID)
		if err != nil {
			continue // libro removido del catálogo; lo saltamos
		}
		posicion++
		etiqueta := "préstamos"
		if e.veces == 1 {
			etiqueta = "préstamo"
		}
		sb.WriteString(fmt.Sprintf("  %d. %s — %s (%d %s)\n",
			posicion, libro.Titulo(), libro.Autor(), e.veces, etiqueta))
	}
	if posicion == 0 {
		return "No hay libros vigentes con préstamos registrados.\n"
	}
	return sb.String()
}
