// Package reservas implementa la cola de espera cuando un libro no está
// disponible. Cuando el usuario intenta prestar un libro prestado, en vez
// de recibir un error puede unirse a una cola FIFO; cuando el libro se
// devuelve, el primero en la cola queda notificado.
//
// Este paquete implementa la rama "libro no disponible" del diagrama de
// flujo del sistema (Figura 5 de la planeación de la Etapa 1) usando una
// estructura de datos clásica de la asignatura: la cola.
package reservas

import (
	"sync"

	"github.com/ronnycort/leelibre/internal/errores"
)

// Cola implementa una cola FIFO (First In, First Out) de usuarios esperando
// un libro específico. Internamente usa un slice, pero la interface pública
// es la de una cola pura: solo se puede Encolar (al final), Desencolar (del
// frente) y Consultar el frente.
type Cola struct {
	mu        sync.RWMutex
	elementos []int // ids de usuarios en orden de llegada
}

// NewCola construye una cola vacía.
func NewCola() *Cola {
	return &Cola{
		elementos: make([]int, 0),
	}
}

// Encolar agrega un usuario al final de la cola. Retorna error si el usuario
// ya está en la cola, para evitar reservas duplicadas.
// La comprobación de duplicados y el append comparten un único Lock, para que
// dos peticiones simultáneas del mismo usuario no puedan colarse las dos.
// Por eso llama a contiene (sin candado) y no a Contiene: sync.RWMutex no es
// reentrante y pedirlo dos veces desde la misma goroutina bloquearía el
// programa.
func (c *Cola) Encolar(usuarioID int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.contiene(usuarioID) {
		return errores.ErrYaEnCola
	}
	c.elementos = append(c.elementos, usuarioID)
	return nil
}

// Desencolar retira y retorna el primer usuario de la cola.
// Retorna error si la cola está vacía.
func (c *Cola) Desencolar() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.elementos) == 0 {
		return 0, errores.ErrColaVacia
	}
	primero := c.elementos[0]
	c.elementos = c.elementos[1:]
	return primero, nil
}

// Frente retorna el primer usuario de la cola sin retirarlo.
// Útil para saber quién será el próximo en recibir el libro.
func (c *Cola) Frente() (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.elementos) == 0 {
		return 0, errores.ErrColaVacia
	}
	return c.elementos[0], nil
}

// Contiene indica si un usuario ya está en la cola.
func (c *Cola) Contiene(usuarioID int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.contiene(usuarioID)
}

// contiene es la versión sin candado, para usar desde métodos que ya lo
// tienen tomado. Al ser minúscula no sale del paquete.
func (c *Cola) contiene(usuarioID int) bool {
	for _, id := range c.elementos {
		if id == usuarioID {
			return true
		}
	}
	return false
}

// Tamano retorna cuántos usuarios están esperando.
func (c *Cola) Tamano() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.elementos)
}

// EstaVacia indica si no hay nadie esperando.
func (c *Cola) EstaVacia() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.elementos) == 0
}

// Elementos retorna una copia del contenido de la cola, en orden.
// Se retorna una copia para respetar la encapsulación.
func (c *Cola) Elementos() []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copia := make([]int, len(c.elementos))
	copy(copia, c.elementos)
	return copia
}

// GestorReservas administra las colas de reserva por libro. Cada libro
// tiene su propia cola, indexadas en un map por id de libro.
type GestorReservas struct {
	// Protege el map de colas. Cada Cola tiene además su propio candado: el
	// orden de toma es siempre gestor y después cola, nunca al revés, para
	// que no puedan quedar dos goroutines esperándose mutuamente.
	mu            sync.RWMutex
	colasPorLibro map[int]*Cola
}

// NewGestorReservas crea un gestor vacío.
func NewGestorReservas() *GestorReservas {
	return &GestorReservas{
		colasPorLibro: make(map[int]*Cola),
	}
}

// Reservar añade a un usuario a la cola de espera de un libro. Si es la
// primera reserva de ese libro, la cola se crea automáticamente.
func (g *GestorReservas) Reservar(libroID, usuarioID int) error {
	g.mu.Lock()
	cola, existe := g.colasPorLibro[libroID]
	if !existe {
		cola = NewCola()
		g.colasPorLibro[libroID] = cola
	}
	g.mu.Unlock()
	// Se suelta el candado del gestor antes de encolar: a partir de aquí ya
	// solo se toca la cola, que se protege sola.
	return cola.Encolar(usuarioID)
}

// LiberarLibro se llama cuando un libro es devuelto. Retira al primer
// usuario de la cola (si hay alguno) para que pueda recibir el libro.
// Retorna el id del usuario que ya puede tomarlo, o 0 si la cola está vacía.
func (g *GestorReservas) LiberarLibro(libroID int) (int, error) {
	g.mu.RLock()
	cola, existe := g.colasPorLibro[libroID]
	g.mu.RUnlock()
	if !existe || cola.EstaVacia() {
		return 0, nil // no hay nadie esperando, no es error
	}
	return cola.Desencolar()
}

// ColaDe retorna la cola completa de un libro (para consultar reservas).
func (g *GestorReservas) ColaDe(libroID int) *Cola {
	g.mu.RLock()
	defer g.mu.RUnlock()
	cola, existe := g.colasPorLibro[libroID]
	if !existe {
		return NewCola() // cola vacía en vez de nil, evita nil-checks
	}
	return cola
}

// PosicionEnCola indica qué posición ocupa un usuario en la cola de un
// libro (1 = primero, 2 = segundo, etc.). Devuelve 0 si no está en cola.
func (g *GestorReservas) PosicionEnCola(libroID, usuarioID int) int {
	cola := g.ColaDe(libroID)
	for i, id := range cola.Elementos() {
		if id == usuarioID {
			return i + 1
		}
	}
	return 0
}

// TotalReservas retorna el total de usuarios esperando en todas las colas.
func (g *GestorReservas) TotalReservas() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	total := 0
	for _, cola := range g.colasPorLibro {
		total += cola.Tamano()
	}
	return total
}
