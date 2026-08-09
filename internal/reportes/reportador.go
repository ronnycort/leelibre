// Package reportes define reportes sobre los datos del sistema.
// Introduce la interface Reportador para demostrar polimorfismo: tres
// tipos distintos (MasPrestados, MasCategorias, Recomendador) implementan
// la misma interface y pueden ser tratados de forma uniforme.
package reportes

// Reportador es la abstracción común de todos los reportes del sistema.
// Cualquier tipo que implemente los dos métodos Nombre() y Generar()
// automáticamente satisface esta interface (Go usa duck typing estático).
//
// Esta es la puerta al polimorfismo: el código que consume Reportador no
// necesita saber si tiene entre manos un reporte de libros más prestados
// o el recomendador personalizado — ambos se usan igual, y en el futuro
// se pueden agregar más implementaciones sin cambiar el código consumidor.
type Reportador interface {
	Nombre() string  // título legible del reporte
	Generar() string // contenido del reporte formateado
}

// EjecutarTodos recibe un slice de Reportador y ejecuta cada uno.
// Es un ejemplo directo de polimorfismo: la función no sabe qué tipos
// concretos recibe, solo confía en que todos cumplen la interface.
func EjecutarTodos(reportes []Reportador) string {
	resultado := ""
	for _, r := range reportes {
		resultado += "\n=== " + r.Nombre() + " ===\n"
		resultado += r.Generar()
		resultado += "\n"
	}
	return resultado
}
