package prestamos

import (
	"sync"

	"github.com/ronnycort/leelibre/internal/errores"
)

// Historial gestiona todos los préstamos del sistema. Ofrece métodos
// para consultarlos por usuario, por libro o en total. Es una entidad
// de dominio (no una vista/reporte) porque mantiene el estado real.
type Historial struct {
	// Protege el slice y el contador de ids frente a las goroutines del
	// servidor HTTP. Sin candado, dos préstamos simultáneos podrían recibir
	// el mismo id o perderse uno de los dos al hacer append a la vez.
	mu        sync.RWMutex
	prestamos []*Prestamo
	proximoID int
}

// NewHistorial crea un historial vacío.
func NewHistorial() *Historial {
	return &Historial{
		prestamos: make([]*Prestamo, 0),
		proximoID: 1,
	}
}

// Registrar añade un préstamo nuevo. Genera automáticamente el id.
// Retorna el préstamo creado para que el llamante pueda referenciarlo.
func (h *Historial) Registrar(usuarioID, libroID int) (*Prestamo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	p, err := NewPrestamo(h.proximoID, usuarioID, libroID)
	if err != nil {
		return nil, err
	}
	h.prestamos = append(h.prestamos, p)
	h.proximoID++
	return p, nil
}

// AgregarExistente incorpora un préstamo ya construido (útil para cargar
// datos históricos desde archivo). Ajusta proximoID si es necesario.
func (h *Historial) AgregarExistente(p *Prestamo) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prestamos = append(h.prestamos, p)
	if p.ID() >= h.proximoID {
		h.proximoID = p.ID() + 1
	}
}

// BuscarPorID retorna el préstamo con el id dado.
func (h *Historial) BuscarPorID(id int) (*Prestamo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.prestamos {
		if p.ID() == id {
			return p, nil
		}
	}
	return nil, errores.ErrPrestamoNoEncontrado
}

// PorUsuario retorna todos los préstamos de un usuario específico.
func (h *Historial) PorUsuario(usuarioID int) []*Prestamo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	resultado := make([]*Prestamo, 0)
	for _, p := range h.prestamos {
		if p.UsuarioID() == usuarioID {
			resultado = append(resultado, p)
		}
	}
	return resultado
}

// PrestamoActivoDeLibro retorna el préstamo activo (no devuelto) de un
// libro específico, si existe. Sirve para saber quién tiene el libro
// actualmente.
func (h *Historial) PrestamoActivoDeLibro(libroID int) (*Prestamo, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.prestamos {
		if p.LibroID() == libroID && p.EstaActivo() {
			return p, nil
		}
	}
	return nil, errores.ErrPrestamoNoEncontrado
}

// Todos retorna una copia de todos los préstamos.
func (h *Historial) Todos() []*Prestamo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	copia := make([]*Prestamo, len(h.prestamos))
	copy(copia, h.prestamos)
	return copia
}

// ActualizarEstados recorre todos los préstamos y actualiza los que
// hayan vencido. Se llamará al inicio del programa.
// Basta con RLock: aquí solo se recorre el slice. Cada préstamo protege su
// propio estado dentro de ActualizarEstado, de modo que el historial no
// necesita bloquear a los demás lectores mientras tanto.
func (h *Historial) ActualizarEstados() {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, p := range h.prestamos {
		p.ActualizarEstado()
	}
}

// Cantidad retorna el número total de préstamos registrados.
func (h *Historial) Cantidad() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.prestamos)
}

// ConteoPorLibro retorna un map de libroID -> cantidad de veces que
// fue prestado. Se usa para el reporte de libros más prestados.
// Los maps son ideales aquí porque proveen acceso O(1) al conteo.
func (h *Historial) ConteoPorLibro() map[int]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conteo := make(map[int]int)
	for _, p := range h.prestamos {
		conteo[p.LibroID()]++
	}
	return conteo
}
