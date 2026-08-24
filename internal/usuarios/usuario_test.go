package usuarios

import (
	"errors"
	"testing"

	errdom "github.com/ronnycort/leelibre/internal/errores"
)

func TestNewUsuarioValidaciones(t *testing.T) {
	casos := []struct {
		nombre    string
		id        int
		nom       string
		correo    string
		password  string
		rol       Rol
		esperaErr bool
	}{
		{"lector válido", 1, "Ana", "ana@leelibre.ec", "clave123", RolLector, false},
		{"administrador válido", 2, "Ronny", "ronny@leelibre.ec", "clave123", RolAdministrador, false},
		{"id inválido", 0, "Ana", "ana@leelibre.ec", "clave123", RolLector, true},
		{"nombre vacío", 1, "", "ana@leelibre.ec", "clave123", RolLector, true},
		{"correo vacío", 1, "Ana", "", "clave123", RolLector, true},
		{"contraseña vacía", 1, "Ana", "ana@leelibre.ec", "", RolLector, true},
		{"rol inexistente", 1, "Ana", "ana@leelibre.ec", "clave123", Rol("superusuario"), true},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := NewUsuario(c.id, c.nom, c.correo, c.password, c.rol)
			if c.esperaErr && err == nil {
				t.Error("se esperaba un error y no hubo ninguno")
			}
			if !c.esperaErr && err != nil {
				t.Errorf("no se esperaba error, se obtuvo %v", err)
			}
		})
	}
}

func TestEsAdministrador(t *testing.T) {
	lector, _ := NewUsuario(1, "Ana", "ana@leelibre.ec", "clave", RolLector)
	admin, _ := NewUsuario(2, "Ronny", "ronny@leelibre.ec", "clave", RolAdministrador)

	if lector.EsAdministrador() {
		t.Error("un lector no debería ser administrador")
	}
	if !admin.EsAdministrador() {
		t.Error("el administrador no fue reconocido como tal")
	}
}

// TestAutenticarNoRevelaSiElCorreoExiste comprueba una decisión de seguridad:
// el error debe ser el mismo cuando el correo no existe y cuando la
// contraseña es incorrecta. Si fueran distintos, cualquiera podría averiguar
// qué correos están registrados probando uno por uno.
func TestAutenticarNoRevelaSiElCorreoExiste(t *testing.T) {
	a := NewAutenticador()
	u, _ := NewUsuario(1, "Ana", "ana@leelibre.ec", "correcta", RolLector)
	if err := a.Registrar(u); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}

	_, errInexistente := a.Autenticar("nadie@leelibre.ec", "loquesea")
	_, errPassword := a.Autenticar("ana@leelibre.ec", "incorrecta")

	if errInexistente == nil || errPassword == nil {
		t.Fatal("ambos intentos deberían fallar")
	}
	if errInexistente.Error() != errPassword.Error() {
		t.Errorf("los mensajes de error son distintos y filtran información:\n  inexistente: %v\n  contraseña:  %v",
			errInexistente, errPassword)
	}
	if !errors.Is(errPassword, errdom.ErrCredencialesInvalidas) {
		t.Errorf("se esperaba ErrCredencialesInvalidas, se obtuvo %v", errPassword)
	}
}

func TestAutenticarConCredencialesCorrectas(t *testing.T) {
	a := NewAutenticador()
	u, _ := NewUsuario(1, "Ana", "ana@leelibre.ec", "correcta", RolLector)
	a.Registrar(u)

	obtenido, err := a.Autenticar("ana@leelibre.ec", "correcta")
	if err != nil {
		t.Fatalf("no se pudo autenticar con credenciales correctas: %v", err)
	}
	if obtenido.ID() != 1 {
		t.Errorf("se autenticó al usuario equivocado: id %d", obtenido.ID())
	}
}

func TestRegistrarRechazaCorreoRepetido(t *testing.T) {
	a := NewAutenticador()
	primero, _ := NewUsuario(1, "Ana", "ana@leelibre.ec", "clave", RolLector)
	segundo, _ := NewUsuario(2, "Otra Ana", "ana@leelibre.ec", "clave", RolLector)

	if err := a.Registrar(primero); err != nil {
		t.Fatalf("preparación fallida: %v", err)
	}
	if err := a.Registrar(segundo); err == nil {
		t.Error("se aceptaron dos usuarios con el mismo correo")
	}
}

// TestPasswordNoTieneGetter no comprueba comportamiento sino diseño: deja
// constancia de que la contraseña solo se puede verificar, nunca leer. Si
// alguien añadiera un getter en el futuro, el propósito de esta prueba
// recordaría por qué no debe haberlo.
func TestPasswordSoloSePuedeVerificar(t *testing.T) {
	u, _ := NewUsuario(1, "Ana", "ana@leelibre.ec", "secreta", RolLector)

	if !u.VerificarPassword("secreta") {
		t.Error("la contraseña correcta no fue aceptada")
	}
	if u.VerificarPassword("otra") {
		t.Error("se aceptó una contraseña incorrecta")
	}
}
