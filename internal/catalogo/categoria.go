// Package catalogo agrupa las entidades relacionadas con los libros:
// Libro, Categoría y el propio Catálogo que los administra.
package catalogo

import "github.com/ronnycort/leelibre/internal/errores"

// Categoria representa una clasificación de libros (Ficción, Ciencia, etc.).
// Los campos son privados y solo se acceden mediante getters, aplicando
// encapsulación por convención de Go (minúscula = privado del paquete).
type Categoria struct {
	id     int
	nombre string
}

// NewCategoria es el constructor de Categoria. Valida los datos antes de
// construir el valor y retorna un error si algo no es válido.
// Devolver el error en el constructor permite fallar temprano.
func NewCategoria(id int, nombre string) (*Categoria, error) {
	if id <= 0 || nombre == "" {
		return nil, errores.ErrDatosInvalidos
	}
	return &Categoria{
		id:     id,
		nombre: nombre,
	}, nil
}

// ID es el getter idiomático del identificador. En Go no se antepone "Get"
// al nombre; el propio nombre del campo con mayúscula ya indica que es un
// getter público.
func (c *Categoria) ID() int {
	return c.id
}

// Nombre retorna el nombre de la categoría.
func (c *Categoria) Nombre() string {
	return c.nombre
}
