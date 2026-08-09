package catalogo

import (
	"fmt"
	"sync"

	"github.com/ronnycort/leelibre/internal/errores"
)

// Formato representa el formato digital del libro. Se define como un tipo
// propio (no un string libre) para restringir valores válidos.
type Formato string

const (
	FormatoPDF  Formato = "PDF"
	FormatoEPUB Formato = "EPUB"
)

// EsValido verifica si el formato es uno de los aceptados.
// Es un método de tipo (no de puntero) porque solo consulta.
func (f Formato) EsValido() bool {
	return f == FormatoPDF || f == FormatoEPUB
}

// Libro representa un libro del catálogo. Todos sus campos son privados
// para forzar el acceso mediante getters y setters controlados. Esto es
// encapsulación pura: nadie fuera del paquete puede modificar el estado
// del libro sin pasar por los métodos que este define.
type Libro struct {
	id          int
	titulo      string
	autor       string
	categoriaID int
	anio        int
	formato     Formato

	// disponible es el único campo que cambia después de construir el libro,
	// así que es el único que necesita protección: el servidor HTTP atiende
	// cada petición en su propia goroutine y varias pueden prestar o devolver
	// el mismo libro a la vez. Los demás campos son inmutables tras el
	// constructor y por eso sus getters no toman el candado.
	mu         sync.RWMutex
	disponible bool
}

// NewLibro es el constructor. Retorna un puntero al libro creado o un
// error si los datos no cumplen las validaciones mínimas.
func NewLibro(id int, titulo, autor string, categoriaID, anio int, formato Formato) (*Libro, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id debe ser positivo", errores.ErrDatosInvalidos)
	}
	if titulo == "" || autor == "" {
		return nil, fmt.Errorf("%w: título y autor son obligatorios", errores.ErrDatosInvalidos)
	}
	if anio < 1000 || anio > 2100 {
		return nil, fmt.Errorf("%w: año fuera de rango", errores.ErrDatosInvalidos)
	}
	if !formato.EsValido() {
		return nil, fmt.Errorf("%w: formato debe ser PDF o EPUB", errores.ErrDatosInvalidos)
	}
	return &Libro{
		id:          id,
		titulo:      titulo,
		autor:       autor,
		categoriaID: categoriaID,
		anio:        anio,
		formato:     formato,
		disponible:  true, // por defecto el libro está disponible
	}, nil
}

// Getters idiomáticos: sin prefijo "Get", solo el nombre del campo con
// mayúscula inicial. Todos usan value receiver excepto cuando podrían
// beneficiarse de un puntero por eficiencia. Como Libro es pequeño,
// usamos punteros solo por consistencia con los métodos modificadores.

// ID retorna el identificador del libro.
func (l *Libro) ID() int { return l.id }

// Titulo retorna el título del libro.
func (l *Libro) Titulo() string { return l.titulo }

// Autor retorna el nombre del autor.
func (l *Libro) Autor() string { return l.autor }

// CategoriaID retorna el identificador de la categoría a la que pertenece.
func (l *Libro) CategoriaID() int { return l.categoriaID }

// Anio retorna el año de publicación.
func (l *Libro) Anio() int { return l.anio }

// Formato retorna el formato del libro.
func (l *Libro) Formato() Formato { return l.formato }

// Disponible indica si el libro puede ser prestado en este momento.
// Toma el candado en modo lectura: varias goroutines pueden consultar a la
// vez, pero ninguna lo hará mientras Prestar o Devolver estén escribiendo.
func (l *Libro) Disponible() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.disponible
}

// Prestar marca el libro como no disponible. Usa pointer receiver porque
// modifica el estado interno del libro; con value receiver se trabajaría
// sobre una copia y el cambio no se reflejaría en el original.
// La comprobación y la escritura van dentro del mismo Lock a propósito: si
// se consultara Disponible() y luego se escribiera en dos pasos separados,
// dos goroutines podrían leer "disponible" a la vez y prestar ambas el mismo
// libro.
func (l *Libro) Prestar() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.disponible {
		return errores.ErrLibroNoDisponible
	}
	l.disponible = false
	return nil
}

// Devolver marca el libro como disponible de nuevo. También es pointer
// receiver por el mismo motivo que Prestar.
func (l *Libro) Devolver() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.disponible = true
}

// String implementa la interface fmt.Stringer, lo que permite imprimir
// el libro directamente con fmt.Println. Esta es la primera aparición
// de polimorfismo: cualquier tipo que implemente String() se comporta
// como un Stringer.
func (l *Libro) String() string {
	estado := "disponible"
	if !l.Disponible() {
		estado = "prestado"
	}
	return fmt.Sprintf("[%d] %s — %s (%d, %s) — %s",
		l.id, l.titulo, l.autor, l.anio, l.formato, estado)
}
