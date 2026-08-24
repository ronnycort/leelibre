package catalogo

import (
	"errors"
	"testing"

	errdom "github.com/ronnycort/leelibre/internal/errores"
)

// catalogoDePrueba arma un catálogo pequeño y conocido. Tener este ayudante
// evita repetir la misma preparación en cada prueba y, sobre todo, que una
// prueba dependa del contenido de data/libros.json: si mañana cambian los
// datos de ejemplo, estas pruebas siguen valiendo.
func catalogoDePrueba(t *testing.T) *Catalogo {
	t.Helper() // marca esta función como ayudante: los fallos se reportan en la línea que la llamó

	c := NewCatalogo()
	ficcion, _ := NewCategoria(1, "Ficción")
	ciencia, _ := NewCategoria(2, "Ciencia")
	if err := c.AgregarCategoria(ficcion); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}
	if err := c.AgregarCategoria(ciencia); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}

	libros := []struct {
		id     int
		titulo string
		autor  string
		catID  int
	}{
		{1, "Rayuela", "Julio Cortázar", 1},
		{2, "Cosmos", "Carl Sagan", 2},
		{3, "Ficciones", "Jorge Luis Borges", 1},
	}
	for _, l := range libros {
		libro, err := NewLibro(l.id, l.titulo, l.autor, l.catID, 1980, FormatoPDF)
		if err != nil {
			t.Fatalf("preparación fallida: %v", err)
		}
		if err := c.AgregarLibro(libro); err != nil {
			t.Fatalf("preparación fallida: %v", err)
		}
	}
	return c
}

func TestAgregarLibroRechazaDuplicados(t *testing.T) {
	c := catalogoDePrueba(t)

	repetido, _ := NewLibro(1, "Otro título", "Otro autor", 1, 2000, FormatoPDF)
	if err := c.AgregarLibro(repetido); err == nil {
		t.Error("se aceptó un libro con un id que ya existía")
	}
	if c.Cantidad() != 3 {
		t.Errorf("el catálogo cambió de tamaño: se esperaban 3 libros, hay %d", c.Cantidad())
	}
}

func TestAgregarLibroRechazaCategoriaInexistente(t *testing.T) {
	c := catalogoDePrueba(t)

	huerfano, _ := NewLibro(99, "Sin categoría", "Autora", 77, 2000, FormatoPDF)
	err := c.AgregarLibro(huerfano)
	if err == nil {
		t.Fatal("se aceptó un libro de una categoría que no existe")
	}
	if !errors.Is(err, errdom.ErrDatosInvalidos) {
		t.Errorf("se esperaba ErrDatosInvalidos, se obtuvo %v", err)
	}
}

// TestBuscarPorTexto comprueba la búsqueda insensible a mayúsculas y que
// busque tanto en el título como en el autor.
func TestBuscarPorTexto(t *testing.T) {
	c := catalogoDePrueba(t)

	casos := []struct {
		nombre    string
		texto     string
		esperados int
	}{
		{"por título exacto", "Cosmos", 1},
		{"sin distinguir mayúsculas", "cosmos", 1},
		{"por nombre del autor", "Borges", 1},
		{"coincidencia parcial", "Cort", 1},
		{"sin coincidencias", "zzzz", 0},
		{"texto vacío devuelve todo", "", 3},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			r := c.BuscarPorTexto(caso.texto)
			if len(r) != caso.esperados {
				t.Errorf("se esperaban %d resultados, se obtuvieron %d", caso.esperados, len(r))
			}
		})
	}
}

// TestFiltrarConPredicado comprueba que Filtrar acepta cualquier condición.
// Es el caso donde se ve que en Go las funciones son valores: la condición se
// pasa como parámetro igual que se pasaría un número.
func TestFiltrarConPredicado(t *testing.T) {
	c := catalogoDePrueba(t)

	deFiccion := c.Filtrar(func(l *Libro) bool { return l.CategoriaID() == 1 })
	if len(deFiccion) != 2 {
		t.Errorf("se esperaban 2 libros de ficción, se obtuvieron %d", len(deFiccion))
	}

	ninguno := c.Filtrar(func(l *Libro) bool { return false })
	if len(ninguno) != 0 {
		t.Errorf("un predicado que siempre es falso debería devolver 0, devolvió %d", len(ninguno))
	}
}

// TestTodosDevuelveCopia protege la encapsulación: si Todos() devolviera el
// slice interno, quien lo recibe podría vaciarlo y romper el catálogo desde
// fuera sin pasar por ningún método.
func TestTodosDevuelveCopia(t *testing.T) {
	c := catalogoDePrueba(t)

	copia := c.Todos()
	copia = copia[:0] // vaciar la copia recibida
	_ = copia

	if c.Cantidad() != 3 {
		t.Errorf("modificar el slice devuelto afectó al catálogo: quedan %d libros", c.Cantidad())
	}
}

func TestEliminarLibro(t *testing.T) {
	c := catalogoDePrueba(t)

	if err := c.EliminarLibro(2); err != nil {
		t.Fatalf("no se pudo eliminar un libro existente: %v", err)
	}
	if c.Cantidad() != 2 {
		t.Errorf("se esperaban 2 libros tras eliminar, hay %d", c.Cantidad())
	}
	if _, err := c.BuscarPorID(2); !errors.Is(err, errdom.ErrLibroNoEncontrado) {
		t.Error("el libro eliminado sigue apareciendo en la búsqueda")
	}
	if err := c.EliminarLibro(999); !errors.Is(err, errdom.ErrLibroNoEncontrado) {
		t.Error("eliminar un libro inexistente debería devolver ErrLibroNoEncontrado")
	}
}

func TestDisponiblesExcluyePrestados(t *testing.T) {
	c := catalogoDePrueba(t)

	libro, _ := c.BuscarPorID(1)
	if err := libro.Prestar(); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}

	if len(c.Disponibles()) != 2 {
		t.Errorf("se esperaban 2 libros disponibles, hay %d", len(c.Disponibles()))
	}
}
