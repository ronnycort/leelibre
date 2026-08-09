// Package usuarios administra los usuarios del sistema y su autenticación.
package usuarios

import (
	"fmt"

	"github.com/ronnycort/leelibre/internal/errores"
)

// Rol define el tipo de usuario. Se implementa como un tipo string con
// constantes para restringir los valores válidos. Esto evita bugs
// difíciles de encontrar cuando alguien pasa "admin" en vez de "Admin".
type Rol string

const (
	RolLector        Rol = "lector"
	RolAdministrador Rol = "administrador"
)

// EsValido verifica si el rol es uno de los conocidos.
func (r Rol) EsValido() bool {
	return r == RolLector || r == RolAdministrador
}

// Usuario representa a una persona registrada en el sistema.
// Todos los campos son privados; en particular la contraseña nunca
// puede ser leída desde fuera del paquete (no hay getter para ella),
// solo puede verificarse mediante VerificarPassword.
type Usuario struct {
	id       int
	nombre   string
	correo   string
	password string
	rol      Rol
}

// NewUsuario es el constructor. Aplica todas las validaciones antes
// de construir el usuario. Como el constructor puede fallar, retorna
// (*Usuario, error) — patrón muy común en Go.
func NewUsuario(id int, nombre, correo, password string, rol Rol) (*Usuario, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id inválido", errores.ErrDatosInvalidos)
	}
	if nombre == "" || correo == "" || password == "" {
		return nil, fmt.Errorf("%w: nombre, correo y contraseña son obligatorios",
			errores.ErrDatosInvalidos)
	}
	if !rol.EsValido() {
		return nil, fmt.Errorf("%w: rol desconocido %q", errores.ErrDatosInvalidos, rol)
	}
	return &Usuario{
		id:       id,
		nombre:   nombre,
		correo:   correo,
		password: password,
		rol:      rol,
	}, nil
}

// ID retorna el identificador del usuario.
func (u *Usuario) ID() int { return u.id }

// Nombre retorna el nombre del usuario.
func (u *Usuario) Nombre() string { return u.nombre }

// Correo retorna el correo del usuario.
func (u *Usuario) Correo() string { return u.correo }

// Rol retorna el rol del usuario.
func (u *Usuario) Rol() Rol { return u.rol }

// EsAdministrador indica si el usuario tiene rol de administrador.
// Aunque podríamos comparar directamente con Rol(), tener este método
// hace el código llamante más legible: u.EsAdministrador() se lee mejor
// que u.Rol() == usuarios.RolAdministrador.
func (u *Usuario) EsAdministrador() bool {
	return u.rol == RolAdministrador
}

// VerificarPassword compara la contraseña dada contra la del usuario.
// No expone la contraseña real, solo confirma si coincide.
// En un sistema real usaríamos bcrypt aquí; para esta demo académica
// una comparación directa es suficiente.
func (u *Usuario) VerificarPassword(password string) bool {
	return u.password == password
}
