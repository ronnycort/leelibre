package usuarios

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ronnycort/leelibre/internal/errores"
)

// Autenticador gestiona el conjunto de usuarios registrados y valida
// las credenciales de acceso al sistema.
type Autenticador struct {
	usuarios  map[int]*Usuario    // índice por id para consultas rápidas
	porCorreo map[string]*Usuario // índice adicional por correo para login
}

// NewAutenticador crea un autenticador vacío.
func NewAutenticador() *Autenticador {
	return &Autenticador{
		usuarios:  make(map[int]*Usuario),
		porCorreo: make(map[string]*Usuario),
	}
}

// Registrar añade un usuario al autenticador.
func (a *Autenticador) Registrar(u *Usuario) error {
	if _, existe := a.usuarios[u.ID()]; existe {
		return fmt.Errorf("%w: ya existe un usuario con id %d",
			errores.ErrDatosInvalidos, u.ID())
	}
	if _, existe := a.porCorreo[u.Correo()]; existe {
		return fmt.Errorf("%w: el correo %s ya está registrado",
			errores.ErrDatosInvalidos, u.Correo())
	}
	a.usuarios[u.ID()] = u
	a.porCorreo[u.Correo()] = u
	return nil
}

// Autenticar valida un par correo/contraseña y devuelve el usuario si
// las credenciales son correctas. Devuelve el mismo error para "correo
// inexistente" y "contraseña incorrecta" para no filtrar información
// (buena práctica de seguridad).
func (a *Autenticador) Autenticar(correo, password string) (*Usuario, error) {
	u, existe := a.porCorreo[correo]
	if !existe {
		return nil, errores.ErrCredencialesInvalidas
	}
	if !u.VerificarPassword(password) {
		return nil, errores.ErrCredencialesInvalidas
	}
	return u, nil
}

// BuscarPorID retorna un usuario por su id.
func (a *Autenticador) BuscarPorID(id int) (*Usuario, error) {
	u, existe := a.usuarios[id]
	if !existe {
		return nil, errores.ErrUsuarioNoEncontrado
	}
	return u, nil
}

// Todos devuelve todos los usuarios registrados.
func (a *Autenticador) Todos() []*Usuario {
	lista := make([]*Usuario, 0, len(a.usuarios))
	for _, u := range a.usuarios {
		lista = append(lista, u)
	}
	return lista
}

// ---------------------------------------------------------------------
// Carga desde archivo JSON
// ---------------------------------------------------------------------

type usuarioJSON struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Correo   string `json:"correo"`
	Password string `json:"password"`
	Rol      string `json:"rol"`
}

// CargarDesdeArchivo carga los usuarios desde un archivo JSON.
func (a *Autenticador) CargarDesdeArchivo(ruta string) error {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", ruta, err)
	}
	var lista []usuarioJSON
	if err := json.Unmarshal(datos, &lista); err != nil {
		return fmt.Errorf("JSON inválido en %s: %w", ruta, err)
	}
	for _, uj := range lista {
		u, err := NewUsuario(uj.ID, uj.Nombre, uj.Correo, uj.Password, Rol(uj.Rol))
		if err != nil {
			return fmt.Errorf("usuario %d inválido: %w", uj.ID, err)
		}
		if err := a.Registrar(u); err != nil {
			return err
		}
	}
	return nil
}
