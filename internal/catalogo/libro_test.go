package catalogo

import (
	"errors"
	"testing"

	errdom "github.com/ronnycort/leelibre/internal/errores"
)

// TestNewLibroValidaciones usa el patrón de pruebas basadas en tabla: en vez
// de escribir una función de prueba por caso, se define un slice con las
// entradas y el resultado esperado, y se recorre llamando a t.Run por cada
// uno. Así, al agregar un caso nuevo solo se añade una fila.
//
// La ventaja de t.Run es que cada caso aparece con su propio nombre en la
// salida: si falla "año fuera de rango", se sabe exactamente cuál falló sin
// tener que leer el código.
func TestNewLibroValidaciones(t *testing.T) {
	casos := []struct {
		nombre    string
		id        int
		titulo    string
		autor     string
		anio      int
		formato   Formato
		esperaErr bool
	}{
		{"libro válido", 1, "Cosmos", "Carl Sagan", 1980, FormatoPDF, false},
		{"id cero", 0, "Cosmos", "Carl Sagan", 1980, FormatoPDF, true},
		{"id negativo", -5, "Cosmos", "Carl Sagan", 1980, FormatoPDF, true},
		{"título vacío", 1, "", "Carl Sagan", 1980, FormatoPDF, true},
		{"autor vacío", 1, "Cosmos", "", 1980, FormatoPDF, true},
		{"año muy antiguo", 1, "Cosmos", "Carl Sagan", 999, FormatoPDF, true},
		{"año futuro", 1, "Cosmos", "Carl Sagan", 2101, FormatoPDF, true},
		{"formato inexistente", 1, "Cosmos", "Carl Sagan", 1980, Formato("MOBI"), true},
		{"formato EPUB válido", 1, "Cosmos", "Carl Sagan", 1980, FormatoEPUB, false},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			libro, err := NewLibro(c.id, c.titulo, c.autor, 1, c.anio, c.formato)

			if c.esperaErr {
				if err == nil {
					t.Fatalf("se esperaba un error y no hubo ninguno")
				}
				// No basta con que falle: debe fallar con el error del dominio,
				// para que quien lo reciba pueda distinguirlo con errors.Is.
				if !errors.Is(err, errdom.ErrDatosInvalidos) {
					t.Errorf("se esperaba ErrDatosInvalidos, se obtuvo %v", err)
				}
				return
			}

			// t.Fatal en vez de t.Error: si el libro es nil, seguir comprobando
			// sus campos provocaría un pánico y ocultaría el fallo real.
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
			if libro == nil {
				t.Fatal("el libro es nil pese a no haber error")
			}
			if !libro.Disponible() {
				t.Error("un libro recién creado debería estar disponible")
			}
		})
	}
}

// TestSettersRechazanValoresInvalidos comprueba la regla más importante de la
// encapsulación: que un modificador rechace un valor imposible y, sobre todo,
// que el objeto conserve su valor anterior en lugar de quedar a medio cambiar.
func TestSettersRechazanValoresInvalidos(t *testing.T) {
	libro, err := NewLibro(1, "Cosmos", "Carl Sagan", 1, 1980, FormatoPDF)
	if err != nil {
		t.Fatalf("no se pudo construir el libro de prueba: %v", err)
	}

	casos := []struct {
		nombre   string
		aplicar  func() error
		revisar  func() bool
		original string
	}{
		{
			nombre:   "año fuera de rango",
			aplicar:  func() error { return libro.SetAnio(3500) },
			revisar:  func() bool { return libro.Anio() == 1980 },
			original: "1980",
		},
		{
			nombre:   "título vacío",
			aplicar:  func() error { return libro.SetTitulo("   ") },
			revisar:  func() bool { return libro.Titulo() == "Cosmos" },
			original: "Cosmos",
		},
		{
			nombre:   "autor vacío",
			aplicar:  func() error { return libro.SetAutor("") },
			revisar:  func() bool { return libro.Autor() == "Carl Sagan" },
			original: "Carl Sagan",
		},
		{
			nombre:   "formato inexistente",
			aplicar:  func() error { return libro.SetFormato(Formato("MOBI")) },
			revisar:  func() bool { return libro.Formato() == FormatoPDF },
			original: "PDF",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if err := c.aplicar(); err == nil {
				t.Error("el modificador aceptó un valor que debía rechazar")
			}
			if !c.revisar() {
				t.Errorf("el libro no conservó su valor anterior (%s)", c.original)
			}
		})
	}
}

// TestSettersAceptanValoresValidos es la otra mitad: comprobar que rechaza lo
// inválido no sirve de nada si también rechaza lo válido.
func TestSettersAceptanValoresValidos(t *testing.T) {
	libro, _ := NewLibro(1, "Cosmos", "Carl Sagan", 1, 1980, FormatoPDF)

	if err := libro.SetAnio(1990); err != nil {
		t.Errorf("SetAnio rechazó un año válido: %v", err)
	}
	if libro.Anio() != 1990 {
		t.Errorf("el año no se actualizó: se esperaba 1990, hay %d", libro.Anio())
	}
	if err := libro.SetTitulo("Cosmos (edición revisada)"); err != nil {
		t.Errorf("SetTitulo rechazó un título válido: %v", err)
	}
}

// TestPrestarYDevolver verifica la regla de que un libro prestado no puede
// volver a prestarse hasta que se devuelva.
func TestPrestarYDevolver(t *testing.T) {
	libro, _ := NewLibro(1, "Cosmos", "Carl Sagan", 1, 1980, FormatoPDF)

	if err := libro.Prestar(); err != nil {
		t.Fatalf("no se pudo prestar un libro disponible: %v", err)
	}
	if libro.Disponible() {
		t.Error("el libro sigue marcado como disponible después de prestarlo")
	}

	err := libro.Prestar()
	if err == nil {
		t.Error("se permitió prestar dos veces el mismo libro")
	}
	if !errors.Is(err, errdom.ErrLibroNoDisponible) {
		t.Errorf("se esperaba ErrLibroNoDisponible, se obtuvo %v", err)
	}

	libro.Devolver()
	if !libro.Disponible() {
		t.Error("el libro no volvió a estar disponible tras devolverlo")
	}
}
