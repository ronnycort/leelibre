// Package persistencia guarda el estado del sistema en archivos JSON.
// Complementa a los métodos CargarDesdeArchivo de los otros paquetes,
// permitiendo el ciclo completo leer -> modificar -> escribir.
package persistencia

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ronnycort/leelibre/internal/catalogo"
	"github.com/ronnycort/leelibre/internal/errores"
	"github.com/ronnycort/leelibre/internal/prestamos"
)

// libroJSON refleja la forma que tienen los libros en el archivo.
// Se declara aquí (no en catalogo) porque solo la usamos para persistir.
type libroJSON struct {
	ID          int    `json:"id"`
	Titulo      string `json:"titulo"`
	Autor       string `json:"autor"`
	CategoriaID int    `json:"categoria_id"`
	Anio        int    `json:"anio"`
	Formato     string `json:"formato"`
}

// prestamoJSON refleja la forma que tienen los préstamos en el archivo.
type prestamoJSON struct {
	ID               int        `json:"id"`
	UsuarioID        int        `json:"usuario_id"`
	LibroID          int        `json:"libro_id"`
	FechaInicio      time.Time  `json:"fecha_inicio"`
	FechaVencimiento time.Time  `json:"fecha_vencimiento"`
	FechaDevolucion  *time.Time `json:"fecha_devolucion,omitempty"`
	Estado           string     `json:"estado"`
}

// GuardarLibros escribe todos los libros del catálogo al archivo JSON.
// Sobrescribe el contenido existente. Retorna error explicativo si falla.
func GuardarLibros(cat *catalogo.Catalogo, ruta string) error {
	libros := cat.Todos()
	registros := make([]libroJSON, 0, len(libros))
	for _, l := range libros {
		registros = append(registros, libroJSON{
			ID:          l.ID(),
			Titulo:      l.Titulo(),
			Autor:       l.Autor(),
			CategoriaID: l.CategoriaID(),
			Anio:        l.Anio(),
			Formato:     string(l.Formato()),
		})
	}
	return escribirJSON(ruta, registros)
}

// GuardarPrestamos escribe todos los préstamos del historial al archivo JSON.
func GuardarPrestamos(hist *prestamos.Historial, ruta string) error {
	todos := hist.Todos()
	registros := make([]prestamoJSON, 0, len(todos))
	for _, p := range todos {
		var fechaDev *time.Time
		if !p.FechaDevolucion().IsZero() {
			f := p.FechaDevolucion()
			fechaDev = &f
		}
		registros = append(registros, prestamoJSON{
			ID:               p.ID(),
			UsuarioID:        p.UsuarioID(),
			LibroID:          p.LibroID(),
			FechaInicio:      p.FechaInicio(),
			FechaVencimiento: p.FechaVencimiento(),
			FechaDevolucion:  fechaDev,
			Estado:           string(p.Estado()),
		})
	}
	return escribirJSON(ruta, registros)
}

// escribirJSON serializa el valor y lo escribe al archivo con formato legible
// (indentado). La escritura se hace de forma atómica: primero a un archivo
// temporal y luego rename, para evitar dejar el archivo corrupto si falla
// en medio de la escritura.
func escribirJSON(ruta string, valor interface{}) error {
	datos, err := json.MarshalIndent(valor, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: al serializar: %v", errores.ErrPersistencia, err)
	}
	rutaTmp := ruta + ".tmp"
	if err := os.WriteFile(rutaTmp, datos, 0644); err != nil {
		return fmt.Errorf("%w: al escribir %s: %v", errores.ErrPersistencia, rutaTmp, err)
	}
	if err := os.Rename(rutaTmp, ruta); err != nil {
		return fmt.Errorf("%w: al renombrar a %s: %v", errores.ErrPersistencia, ruta, err)
	}
	return nil
}
