package catalogo

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/ronnycort/leelibre/internal/errores"
)

// Catalogo agrupa una colección de libros y una colección de categorías.
// Es responsable de las operaciones de alta, baja, búsqueda y filtrado.
// Los slices y maps internos son privados para que nadie los modifique
// desde fuera sin pasar por los métodos que garantizan consistencia.
type Catalogo struct {
	// El servidor HTTP atiende cada petición en una goroutine distinta, así
	// que dos peticiones pueden agregar o eliminar libros a la vez. Un slice
	// y un map no son seguros para uso concurrente, por lo que todo acceso
	// pasa por este candado: RLock para consultar (varios lectores a la vez)
	// y Lock para modificar (un único escritor, sin lectores).
	mu         sync.RWMutex
	libros     []*Libro
	categorias map[int]*Categoria // índice por id para búsquedas rápidas
}

// NewCatalogo crea un catálogo vacío listo para recibir libros y categorías.
func NewCatalogo() *Catalogo {
	return &Catalogo{
		libros:     make([]*Libro, 0),
		categorias: make(map[int]*Categoria),
	}
}

// AgregarCategoria incorpora una categoría al catálogo.
// Retorna error si ya existe una con el mismo id.
func (c *Catalogo) AgregarCategoria(cat *Categoria) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, existe := c.categorias[cat.ID()]; existe {
		return fmt.Errorf("%w: ya existe una categoría con id %d",
			errores.ErrDatosInvalidos, cat.ID())
	}
	c.categorias[cat.ID()] = cat
	return nil
}

// AgregarLibro incorpora un libro al catálogo. Verifica que la categoría
// del libro exista antes de aceptarlo. Este es un ejemplo de invariante
// del dominio: un libro no puede pertenecer a una categoría inexistente.
func (c *Catalogo) AgregarLibro(l *Libro) error {
	// La verificación de duplicados y el append van bajo el mismo Lock: si se
	// hicieran por separado, dos peticiones simultáneas podrían comprobar que
	// el id no existe y añadirlo las dos.
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, existe := c.categorias[l.CategoriaID()]; !existe {
		return fmt.Errorf("%w: la categoría %d no existe",
			errores.ErrDatosInvalidos, l.CategoriaID())
	}
	// Verificar que no exista otro libro con el mismo id
	for _, existente := range c.libros {
		if existente.ID() == l.ID() {
			return fmt.Errorf("%w: ya existe un libro con id %d",
				errores.ErrDatosInvalidos, l.ID())
		}
	}
	c.libros = append(c.libros, l)
	return nil
}

// EliminarLibro remueve un libro del catálogo por su id.
func (c *Catalogo) EliminarLibro(id int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, l := range c.libros {
		if l.ID() == id {
			// Reemplazar el elemento con el último y acortar el slice.
			// Es más eficiente que remover en el medio con desplazamiento.
			c.libros[i] = c.libros[len(c.libros)-1]
			c.libros = c.libros[:len(c.libros)-1]
			return nil
		}
	}
	return errores.ErrLibroNoEncontrado
}

// BuscarPorID retorna un libro dado su identificador o error si no existe.
func (c *Catalogo) BuscarPorID(id int) (*Libro, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, l := range c.libros {
		if l.ID() == id {
			return l, nil
		}
	}
	return nil, errores.ErrLibroNoEncontrado
}

// CategoriaDe retorna la categoría a la que pertenece el libro dado.
func (c *Catalogo) CategoriaDe(l *Libro) (*Categoria, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cat, existe := c.categorias[l.CategoriaID()]
	if !existe {
		return nil, errores.ErrDatosInvalidos
	}
	return cat, nil
}

// Filtrar aplica un predicado a cada libro y devuelve solo los que lo cumplen.
// Recibe una función como parámetro, aprovechando que en Go las funciones
// son valores de primera clase. Es un método que expone una operación
// flexible sin comprometer la encapsulación del slice interno.
// Es el único punto que recorre el slice al filtrar, así que basta con que
// tome aquí el candado: BuscarPorTexto, PorCategoria y Disponibles se apoyan
// en él y por eso no vuelven a pedirlo (RWMutex no es reentrante y pedirlo
// dos veces desde la misma goroutina podría bloquear el programa).
func (c *Catalogo) Filtrar(cumple func(*Libro) bool) []*Libro {
	c.mu.RLock()
	defer c.mu.RUnlock()
	resultado := make([]*Libro, 0)
	for _, l := range c.libros {
		if cumple(l) {
			resultado = append(resultado, l)
		}
	}
	return resultado
}

// BuscarPorTexto busca libros cuyo título o autor contengan el texto dado
// (búsqueda insensible a mayúsculas). Usa Filtrar internamente para
// mostrar composición de métodos.
func (c *Catalogo) BuscarPorTexto(texto string) []*Libro {
	texto = strings.ToLower(strings.TrimSpace(texto))
	return c.Filtrar(func(l *Libro) bool {
		return strings.Contains(strings.ToLower(l.Titulo()), texto) ||
			strings.Contains(strings.ToLower(l.Autor()), texto)
	})
}

// PorCategoria devuelve todos los libros de una categoría dada.
func (c *Catalogo) PorCategoria(catID int) []*Libro {
	return c.Filtrar(func(l *Libro) bool {
		return l.CategoriaID() == catID
	})
}

// Disponibles devuelve solo los libros disponibles para préstamo.
func (c *Catalogo) Disponibles() []*Libro {
	return c.Filtrar(func(l *Libro) bool {
		return l.Disponible()
	})
}

// Todos devuelve una copia del slice de libros. Se retorna una copia
// para respetar la encapsulación: si retornáramos el slice interno,
// el llamante podría reordenarlo o mutarlo.
func (c *Catalogo) Todos() []*Libro {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copia := make([]*Libro, len(c.libros))
	copy(copia, c.libros)
	return copia
}

// TodasCategorias devuelve una copia de las categorías disponibles.
func (c *Catalogo) TodasCategorias() []*Categoria {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cats := make([]*Categoria, 0, len(c.categorias))
	for _, cat := range c.categorias {
		cats = append(cats, cat)
	}
	return cats
}

// Cantidad retorna el número total de libros en el catálogo.
func (c *Catalogo) Cantidad() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.libros)
}

// ---------------------------------------------------------------------
// Carga desde archivos JSON
// ---------------------------------------------------------------------

// libroJSON es una estructura auxiliar usada solo para deserializar.
// Los campos deben ser públicos para que encoding/json pueda leerlos,
// por eso no reutilizamos el struct Libro directamente (sus campos
// son privados para preservar la encapsulación).
type libroJSON struct {
	ID          int    `json:"id"`
	Titulo      string `json:"titulo"`
	Autor       string `json:"autor"`
	CategoriaID int    `json:"categoria_id"`
	Anio        int    `json:"anio"`
	Formato     string `json:"formato"`
}

type categoriaJSON struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}

// CargarDesdeArchivos carga categorías y libros desde archivos JSON.
// Retorna error explicando qué archivo o registro falló, lo cual es
// mucho más útil para depurar que un error genérico.
func (c *Catalogo) CargarDesdeArchivos(rutaCategorias, rutaLibros string) error {
	// Cargar categorías primero (los libros dependen de ellas)
	datosCat, err := os.ReadFile(rutaCategorias)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", rutaCategorias, err)
	}
	var cats []categoriaJSON
	if err := json.Unmarshal(datosCat, &cats); err != nil {
		return fmt.Errorf("JSON inválido en %s: %w", rutaCategorias, err)
	}
	for _, cj := range cats {
		cat, err := NewCategoria(cj.ID, cj.Nombre)
		if err != nil {
			return fmt.Errorf("categoría %d inválida: %w", cj.ID, err)
		}
		if err := c.AgregarCategoria(cat); err != nil {
			return err
		}
	}

	// Cargar libros
	datosLib, err := os.ReadFile(rutaLibros)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", rutaLibros, err)
	}
	var libs []libroJSON
	if err := json.Unmarshal(datosLib, &libs); err != nil {
		return fmt.Errorf("JSON inválido en %s: %w", rutaLibros, err)
	}
	for _, lj := range libs {
		l, err := NewLibro(lj.ID, lj.Titulo, lj.Autor, lj.CategoriaID, lj.Anio, Formato(lj.Formato))
		if err != nil {
			return fmt.Errorf("libro %d inválido: %w", lj.ID, err)
		}
		if err := c.AgregarLibro(l); err != nil {
			return err
		}
	}
	return nil
}
