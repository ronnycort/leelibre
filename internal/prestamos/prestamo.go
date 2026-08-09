// Package prestamos gestiona los préstamos digitales de los libros del
// catálogo por parte de los usuarios registrados.
package prestamos

import (
	"fmt"
	"time"

	"github.com/ronnycort/leelibre/internal/errores"
)

// EstadoPrestamo enumera los estados posibles de un préstamo.
type EstadoPrestamo string

const (
	EstadoVigente  EstadoPrestamo = "vigente"
	EstadoDevuelto EstadoPrestamo = "devuelto"
	EstadoVencido  EstadoPrestamo = "vencido"
)

// Duracion del préstamo por defecto (14 días).
const DiasDePrestamo = 14

// Prestamo representa el permiso otorgado a un usuario para leer un libro
// durante un periodo determinado. Todos los campos son privados.
type Prestamo struct {
	id               int
	usuarioID        int
	libroID          int
	fechaInicio      time.Time
	fechaVencimiento time.Time
	fechaDevolucion  *time.Time // puntero porque puede no existir aún
	estado           EstadoPrestamo
}

// NewPrestamo crea un préstamo nuevo con estado vigente y calcula la
// fecha de vencimiento automáticamente.
func NewPrestamo(id, usuarioID, libroID int) (*Prestamo, error) {
	if id <= 0 || usuarioID <= 0 || libroID <= 0 {
		return nil, fmt.Errorf("%w: ids inválidos", errores.ErrDatosInvalidos)
	}
	ahora := time.Now()
	return &Prestamo{
		id:               id,
		usuarioID:        usuarioID,
		libroID:          libroID,
		fechaInicio:      ahora,
		fechaVencimiento: ahora.AddDate(0, 0, DiasDePrestamo),
		fechaDevolucion:  nil,
		estado:           EstadoVigente,
	}, nil
}

// Getters idiomáticos

func (p *Prestamo) ID() int                     { return p.id }
func (p *Prestamo) UsuarioID() int              { return p.usuarioID }
func (p *Prestamo) LibroID() int                { return p.libroID }
func (p *Prestamo) FechaInicio() time.Time      { return p.fechaInicio }
func (p *Prestamo) FechaVencimiento() time.Time { return p.fechaVencimiento }
func (p *Prestamo) Estado() EstadoPrestamo      { return p.estado }

// FechaDevolucion retorna la fecha de devolución si el préstamo fue
// devuelto, o cero si aún no. Se retorna un valor (no un puntero) para
// no exponer el puntero interno.
func (p *Prestamo) FechaDevolucion() time.Time {
	if p.fechaDevolucion == nil {
		return time.Time{}
	}
	return *p.fechaDevolucion
}

// Devolver marca el préstamo como devuelto en la fecha actual. Retorna
// error si el préstamo ya había sido devuelto. Usa pointer receiver
// porque modifica el estado.
func (p *Prestamo) Devolver() error {
	if p.estado == EstadoDevuelto {
		return fmt.Errorf("%w: el préstamo %d ya fue devuelto",
			errores.ErrDatosInvalidos, p.id)
	}
	ahora := time.Now()
	p.fechaDevolucion = &ahora
	p.estado = EstadoDevuelto
	return nil
}

// ActualizarEstado revisa la fecha actual y marca como vencido el préstamo
// si su fecha de vencimiento ya pasó y sigue vigente. Este método es un
// buen ejemplo de por qué usamos pointer receiver: modifica el estado
// del objeto según una regla del dominio.
func (p *Prestamo) ActualizarEstado() {
	if p.estado == EstadoVigente && time.Now().After(p.fechaVencimiento) {
		p.estado = EstadoVencido
	}
}

// EstaActivo indica si el préstamo sigue en manos del usuario (vigente
// o vencido pero no devuelto).
func (p *Prestamo) EstaActivo() bool {
	return p.estado != EstadoDevuelto
}

// String implementa fmt.Stringer para imprimir el préstamo de forma
// legible en la consola.
func (p *Prestamo) String() string {
	desc := fmt.Sprintf("Préstamo #%d — libro %d, usuario %d — %s — inicio %s, vence %s",
		p.id, p.libroID, p.usuarioID, p.estado,
		p.fechaInicio.Format("2006-01-02"),
		p.fechaVencimiento.Format("2006-01-02"))
	if p.fechaDevolucion != nil {
		desc += fmt.Sprintf(" — devuelto el %s", p.fechaDevolucion.Format("2006-01-02"))
	}
	return desc
}
