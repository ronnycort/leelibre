package prestamos

import (
	"errors"
	"testing"
	"time"

	errdom "github.com/ronnycort/leelibre/internal/errores"
)

func TestNewPrestamoValidaciones(t *testing.T) {
	casos := []struct {
		nombre    string
		id        int
		usuarioID int
		libroID   int
		esperaErr bool
	}{
		{"préstamo válido", 1, 2, 3, false},
		{"id inválido", 0, 2, 3, true},
		{"usuario inválido", 1, 0, 3, true},
		{"libro inválido", 1, 2, 0, true},
		{"todos negativos", -1, -2, -3, true},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p, err := NewPrestamo(c.id, c.usuarioID, c.libroID)
			if c.esperaErr {
				if err == nil {
					t.Error("se esperaba un error y no hubo ninguno")
				}
				return
			}
			if err != nil {
				t.Fatalf("no se esperaba error, se obtuvo %v", err)
			}
			if p.Estado() != EstadoVigente {
				t.Errorf("un préstamo nuevo debería estar vigente, está %q", p.Estado())
			}
		})
	}
}

// TestVencimientoALos14Dias comprueba la regla de negocio del plazo. Aquí se
// ve para qué sirve ActualizarEstadoEn: se le pasa una fecha simulada en vez
// de esperar catorce días reales o cambiar la hora del computador.
func TestVencimientoALos14Dias(t *testing.T) {
	p, err := NewPrestamo(1, 2, 3)
	if err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}

	casos := []struct {
		nombre   string
		momento  time.Time
		esperado EstadoPrestamo
	}{
		{"al día siguiente sigue vigente", time.Now().AddDate(0, 0, 1), EstadoVigente},
		{"el día 13 sigue vigente", time.Now().AddDate(0, 0, 13), EstadoVigente},
		{"el día 15 ya venció", time.Now().AddDate(0, 0, 15), EstadoVencido},
		{"un mes después venció", time.Now().AddDate(0, 1, 0), EstadoVencido},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			// Cada caso parte de un préstamo nuevo: si compartieran el mismo,
			// el primero que lo marcara como vencido afectaría a los siguientes
			// y el resultado dependería del orden de ejecución.
			fresco, _ := NewPrestamo(1, 2, 3)
			fresco.ActualizarEstadoEn(c.momento)
			if fresco.Estado() != c.esperado {
				t.Errorf("se esperaba estado %q, se obtuvo %q", c.esperado, fresco.Estado())
			}
		})
	}
	_ = p
}

func TestDevolverDosVecesFalla(t *testing.T) {
	p, _ := NewPrestamo(1, 2, 3)

	if err := p.Devolver(); err != nil {
		t.Fatalf("no se pudo devolver un préstamo vigente: %v", err)
	}
	if p.Estado() != EstadoDevuelto {
		t.Errorf("se esperaba estado devuelto, se obtuvo %q", p.Estado())
	}
	if p.EstaActivo() {
		t.Error("un préstamo devuelto no debería seguir activo")
	}

	err := p.Devolver()
	if err == nil {
		t.Fatal("se permitió devolver dos veces el mismo préstamo")
	}
	if !errors.Is(err, errdom.ErrDatosInvalidos) {
		t.Errorf("se esperaba ErrDatosInvalidos, se obtuvo %v", err)
	}
}

// TestPrestamoVencidoSigueActivo documenta una regla fácil de malinterpretar:
// vencido no es lo mismo que devuelto. El libro sigue en manos del usuario.
func TestPrestamoVencidoSigueActivo(t *testing.T) {
	p, _ := NewPrestamo(1, 2, 3)
	p.ActualizarEstadoEn(time.Now().AddDate(0, 0, 30))

	if p.Estado() != EstadoVencido {
		t.Fatalf("preparación fallida: el préstamo debería estar vencido, está %q", p.Estado())
	}
	if !p.EstaActivo() {
		t.Error("un préstamo vencido pero no devuelto debería seguir contando como activo")
	}
}

// TestDevueltoNoVence comprueba que el estado no retrocede: una vez devuelto,
// el paso del tiempo ya no puede marcarlo como vencido.
func TestDevueltoNoVence(t *testing.T) {
	p, _ := NewPrestamo(1, 2, 3)
	if err := p.Devolver(); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}

	p.ActualizarEstadoEn(time.Now().AddDate(0, 1, 0))
	if p.Estado() != EstadoDevuelto {
		t.Errorf("un préstamo devuelto cambió a %q al pasar el tiempo", p.Estado())
	}
}

func TestHistorialRegistraYConsulta(t *testing.T) {
	h := NewHistorial()

	if _, err := h.Registrar(1, 10); err != nil {
		t.Fatalf("no se pudo registrar el préstamo: %v", err)
	}
	if _, err := h.Registrar(1, 11); err != nil {
		t.Fatalf("no se pudo registrar el préstamo: %v", err)
	}
	if _, err := h.Registrar(2, 12); err != nil {
		t.Fatalf("no se pudo registrar el préstamo: %v", err)
	}

	if h.Cantidad() != 3 {
		t.Errorf("se esperaban 3 préstamos, hay %d", h.Cantidad())
	}
	if n := len(h.PorUsuario(1)); n != 2 {
		t.Errorf("el usuario 1 debería tener 2 préstamos, tiene %d", n)
	}
	if n := len(h.PorUsuario(99)); n != 0 {
		t.Errorf("un usuario sin préstamos debería devolver 0, devolvió %d", n)
	}
}

// TestHistorialAsignaIdsUnicos comprueba que dos préstamos nunca comparten id.
func TestHistorialAsignaIdsUnicos(t *testing.T) {
	h := NewHistorial()
	vistos := make(map[int]bool)

	for i := 0; i < 20; i++ {
		p, err := h.Registrar(1, i+1)
		if err != nil {
			t.Fatalf("no se pudo registrar el préstamo %d: %v", i, err)
		}
		if vistos[p.ID()] {
			t.Fatalf("el id %d se repitió", p.ID())
		}
		vistos[p.ID()] = true
	}
}

func TestConteoPorLibro(t *testing.T) {
	h := NewHistorial()
	h.Registrar(1, 10)
	h.Registrar(2, 10)
	h.Registrar(3, 10)
	h.Registrar(1, 20)

	conteo := h.ConteoPorLibro()
	if conteo[10] != 3 {
		t.Errorf("el libro 10 debería tener 3 préstamos, tiene %d", conteo[10])
	}
	if conteo[20] != 1 {
		t.Errorf("el libro 20 debería tener 1 préstamo, tiene %d", conteo[20])
	}
	if conteo[99] != 0 {
		t.Errorf("un libro nunca prestado debería dar 0, dio %d", conteo[99])
	}
}
