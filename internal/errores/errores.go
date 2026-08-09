// Package errores contiene los errores personalizados del dominio de LeeLibre.
// Se definen como valores concretos que otras capas pueden retornar y comparar
// mediante errors.Is, lo que permite un manejo de errores explícito y ordenado.
package errores

import "errors"

// Errores de dominio predefinidos. Se exportan (mayúscula inicial) porque
// forman parte del contrato público del paquete.
var (
	ErrLibroNoEncontrado     = errors.New("el libro no existe en el catálogo")
	ErrLibroNoDisponible     = errors.New("el libro no está disponible en este momento")
	ErrUsuarioNoEncontrado   = errors.New("el usuario no existe")
	ErrCredencialesInvalidas = errors.New("correo o contraseña incorrectos")
	ErrPrestamoNoEncontrado  = errors.New("no se encontró el préstamo indicado")
	ErrHistorialVacio        = errors.New("el usuario no tiene historial de lecturas")
	ErrDatosInvalidos        = errors.New("los datos proporcionados no son válidos")
	ErrPermisoDenegado       = errors.New("el usuario no tiene permisos para esta operación")
	ErrYaEnCola              = errors.New("el usuario ya está en la cola de reservas de este libro")
	ErrColaVacia             = errors.New("no hay usuarios en la cola de reservas")
	ErrPersistencia          = errors.New("error al persistir datos")
)
