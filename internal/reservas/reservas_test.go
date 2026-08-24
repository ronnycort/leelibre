package reservas

import (
	"errors"
	"sync"
	"testing"

	errdom "github.com/ronnycort/leelibre/internal/errores"
)

// TestColaRespetaOrdenFIFO comprueba la propiedad que define a una cola: el
// primero que llega es el primero en salir. Es la razón de haber elegido esta
// estructura y no otra para las reservas.
func TestColaRespetaOrdenFIFO(t *testing.T) {
	c := NewCola()

	llegada := []int{7, 3, 9, 1}
	for _, id := range llegada {
		if err := c.Encolar(id); err != nil {
			t.Fatalf("no se pudo encolar al usuario %d: %v", id, err)
		}
	}

	for _, esperado := range llegada {
		obtenido, err := c.Desencolar()
		if err != nil {
			t.Fatalf("no se pudo desencolar: %v", err)
		}
		if obtenido != esperado {
			t.Errorf("se rompió el orden: se esperaba %d, salió %d", esperado, obtenido)
		}
	}
}

func TestColaVaciaDevuelveError(t *testing.T) {
	c := NewCola()

	if !c.EstaVacia() {
		t.Error("una cola recién creada debería estar vacía")
	}
	if _, err := c.Desencolar(); !errors.Is(err, errdom.ErrColaVacia) {
		t.Errorf("se esperaba ErrColaVacia al desencolar, se obtuvo %v", err)
	}
	if _, err := c.Frente(); !errors.Is(err, errdom.ErrColaVacia) {
		t.Errorf("se esperaba ErrColaVacia al consultar el frente, se obtuvo %v", err)
	}
}

// TestNoSePuedeReservarDosVeces evita que un mismo usuario ocupe dos puestos
// en la cola del mismo libro.
func TestNoSePuedeReservarDosVeces(t *testing.T) {
	c := NewCola()

	if err := c.Encolar(5); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}
	err := c.Encolar(5)
	if err == nil {
		t.Fatal("se permitió que el mismo usuario entrara dos veces en la cola")
	}
	if !errors.Is(err, errdom.ErrYaEnCola) {
		t.Errorf("se esperaba ErrYaEnCola, se obtuvo %v", err)
	}
	if c.Tamano() != 1 {
		t.Errorf("la cola debería tener 1 usuario, tiene %d", c.Tamano())
	}
}

// TestFrenteNoRetira distingue Frente de Desencolar: consultar quién es el
// siguiente no debe sacarlo de la fila.
func TestFrenteNoRetira(t *testing.T) {
	c := NewCola()
	c.Encolar(4)
	c.Encolar(8)

	primero, err := c.Frente()
	if err != nil {
		t.Fatalf("no se pudo consultar el frente: %v", err)
	}
	if primero != 4 {
		t.Errorf("el frente debería ser 4, es %d", primero)
	}
	if c.Tamano() != 2 {
		t.Errorf("consultar el frente cambió el tamaño de la cola: %d", c.Tamano())
	}
}

// TestElementosDevuelveCopia protege la encapsulación de la cola.
func TestElementosDevuelveCopia(t *testing.T) {
	c := NewCola()
	c.Encolar(1)
	c.Encolar(2)

	copia := c.Elementos()
	copia[0] = 999

	if frente, _ := c.Frente(); frente == 999 {
		t.Error("modificar el slice devuelto alteró la cola original")
	}
}

func TestPosicionEnCola(t *testing.T) {
	g := NewGestorReservas()

	g.Reservar(100, 1)
	g.Reservar(100, 2)
	g.Reservar(100, 3)

	casos := []struct {
		usuario  int
		esperada int
	}{
		{1, 1},
		{2, 2},
		{3, 3},
		{99, 0}, // quien no está en la cola devuelve 0, no un error
	}

	for _, c := range casos {
		if p := g.PosicionEnCola(100, c.usuario); p != c.esperada {
			t.Errorf("usuario %d: se esperaba la posición %d, se obtuvo %d", c.usuario, c.esperada, p)
		}
	}
}

// TestLiberarLibroAvanzaLaCola comprueba el flujo completo: al devolverse un
// libro, el primero de la fila es quien lo recibe.
func TestLiberarLibroAvanzaLaCola(t *testing.T) {
	g := NewGestorReservas()
	g.Reservar(100, 11)
	g.Reservar(100, 22)

	siguiente, err := g.LiberarLibro(100)
	if err != nil {
		t.Fatalf("no se pudo liberar el libro: %v", err)
	}
	if siguiente != 11 {
		t.Errorf("debería tocarle al usuario 11, le tocó a %d", siguiente)
	}
	if g.PosicionEnCola(100, 22) != 1 {
		t.Error("el usuario 22 debería haber avanzado al primer puesto")
	}
}

// TestLiberarLibroSinColaNoEsError documenta una decisión de diseño: que
// nadie esté esperando un libro es una situación normal, no un fallo.
func TestLiberarLibroSinColaNoEsError(t *testing.T) {
	g := NewGestorReservas()

	siguiente, err := g.LiberarLibro(555)
	if err != nil {
		t.Errorf("liberar un libro sin cola no debería dar error, dio %v", err)
	}
	if siguiente != 0 {
		t.Errorf("sin nadie esperando se debería devolver 0, se devolvió %d", siguiente)
	}
}

// TestReservasConcurrentes ejercita el gestor desde varias goroutines a la
// vez. Con -race, esta prueba falla si falta protección en el map interno.
//
// Comprueba además una propiedad que no depende del orden: da igual quién
// llegue primero, al final los 50 usuarios deben estar en la cola exactamente
// una vez.
func TestReservasConcurrentes(t *testing.T) {
	g := NewGestorReservas()
	const usuarios = 50

	var wg sync.WaitGroup
	for i := 1; i <= usuarios; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			g.Reservar(100, id)
		}(i)
	}
	wg.Wait()

	if total := g.TotalReservas(); total != usuarios {
		t.Errorf("se esperaban %d reservas, hay %d (se perdió alguna por acceso concurrente)", usuarios, total)
	}
}
