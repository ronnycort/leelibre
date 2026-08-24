// Programa principal (CLI) de LeeLibre: Sistema de gestión de biblioteca digital.
// Aplica los conceptos de programación orientada a objetos en Go:
// encapsulación por paquetes, structs con métodos, constructores,
// interfaces con polimorfismo, cola FIFO para reservas y manejo de errores.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ronnycort/leelibre/internal/catalogo"
	errdom "github.com/ronnycort/leelibre/internal/errores"
	"github.com/ronnycort/leelibre/internal/persistencia"
	"github.com/ronnycort/leelibre/internal/prestamos"
	"github.com/ronnycort/leelibre/internal/reportes"
	"github.com/ronnycort/leelibre/internal/reservas"
	"github.com/ronnycort/leelibre/internal/usuarios"
)

const (
	rutaCategorias = "data/categorias.json"
	rutaLibros     = "data/libros.json"
	rutaUsuarios   = "data/usuarios.json"
	rutaPrestamos  = "data/prestamos.json"
)

// App agrupa las referencias a los subsistemas del programa.
type App struct {
	catalogo       *catalogo.Catalogo
	autenticador   *usuarios.Autenticador
	historial      *prestamos.Historial
	gestorReservas *reservas.GestorReservas
	usuarioActual  *usuarios.Usuario // nil cuando no hay sesión
	entrada        *bufio.Scanner
}

// NewApp construye la app cargando los datos iniciales desde JSON.
func NewApp() (*App, error) {
	cat := catalogo.NewCatalogo()
	if err := cat.CargarDesdeArchivos(rutaCategorias, rutaLibros); err != nil {
		return nil, fmt.Errorf("cargando catálogo: %w", err)
	}

	auth := usuarios.NewAutenticador()
	if err := auth.CargarDesdeArchivo(rutaUsuarios); err != nil {
		return nil, fmt.Errorf("cargando usuarios: %w", err)
	}

	hist := prestamos.NewHistorial()
	// Sembrar historial variado para que el recomendador tenga datos
	// y el reporte de más prestados muestre diferencias reales entre libros.
	sembrarHistorial(hist, cat)
	hist.ActualizarEstados()

	return &App{
		catalogo:       cat,
		autenticador:   auth,
		historial:      hist,
		gestorReservas: reservas.NewGestorReservas(),
		entrada:        bufio.NewScanner(os.Stdin),
	}, nil
}

// sembrarHistorial crea préstamos de arranque con cantidades variables
// por libro. Así el reporte de "más prestados" muestra diferencias reales
// en vez de empates aleatorios.
func sembrarHistorial(hist *prestamos.Historial, cat *catalogo.Catalogo) {
	// Formato: (usuario, libro, veces_devuelto)
	// Los "veces_devuelto" simulan préstamos históricos ya cerrados.
	prestamosSemilla := []struct {
		usuario int
		libro   int
		veces   int
	}{
		// Sapiens (libro 11) - MUY popular, 6 préstamos
		{2, 11, 3}, {3, 11, 2}, {4, 11, 1},
		// Cien años de soledad (libro 1) - popular, 4 préstamos
		{2, 1, 2}, {5, 1, 1}, {6, 1, 1},
		// El túnel (libro 2) - popular, 3 préstamos
		{2, 2, 2}, {7, 2, 1},
		// Rayuela (libro 3) - 2 préstamos (perfil de Ana)
		{2, 3, 2},
		// Ciencia para Carlos
		{3, 6, 2}, {3, 7, 1}, {3, 8, 1},
		// Historia/Filosofía para María
		{4, 15, 1}, {4, 16, 1},
		// Tecnología para Juan (2 préstamos)
		{5, 19, 1}, {5, 20, 1},
	}

	for _, ps := range prestamosSemilla {
		for i := 0; i < ps.veces; i++ {
			p, err := hist.Registrar(ps.usuario, ps.libro)
			if err != nil {
				continue
			}
			// Marcar como devuelto para que el libro quede disponible
			// (los préstamos sembrados representan historia ya cerrada).
			_ = p.Devolver()
			if libro, err := cat.BuscarPorID(ps.libro); err == nil {
				libro.Devolver()
			}
		}
	}
}

// guardarCambios persiste los cambios al disco. Se llama después de las
// operaciones que modifican el estado (agregar/eliminar libros, préstamos).
// No es fatal si falla — se avisa pero la operación se completa igual.
func (a *App) guardarCambios() {
	if err := persistencia.GuardarLibros(a.catalogo, rutaLibros); err != nil {
		fmt.Printf("Aviso: no se pudieron guardar los libros: %v\n", err)
	}
	if err := persistencia.GuardarPrestamos(a.historial, rutaPrestamos); err != nil {
		fmt.Printf("Aviso: no se pudieron guardar los préstamos: %v\n", err)
	}
}

// ---------------------------------------------------------------------
// Utilidades de entrada
// ---------------------------------------------------------------------

func (a *App) leerLinea(prompt string) string {
	fmt.Print(prompt)
	if !a.entrada.Scan() {
		return ""
	}
	return strings.TrimSpace(a.entrada.Text())
}

func (a *App) leerEntero(prompt string) (int, error) {
	texto := a.leerLinea(prompt)
	n, err := strconv.Atoi(texto)
	if err != nil {
		return 0, fmt.Errorf("%w: se esperaba un número", errdom.ErrDatosInvalidos)
	}
	return n, nil
}

// tituloLibro es un helper defensivo: retorna el título del libro dado su id,
// o "[libro eliminado #N]" si el libro ya no existe. Este helper evita el
// crash por nil-pointer cuando un libro se elimina del catálogo pero sigue
// referenciado en préstamos históricos.
func (a *App) tituloLibro(libroID int) string {
	libro, err := a.catalogo.BuscarPorID(libroID)
	if err != nil {
		return fmt.Sprintf("[libro eliminado #%d]", libroID)
	}
	return libro.Titulo()
}

// nombreUsuario es el helper equivalente para usuarios.
func (a *App) nombreUsuario(usuarioID int) string {
	u, err := a.autenticador.BuscarPorID(usuarioID)
	if err != nil {
		return fmt.Sprintf("[usuario eliminado #%d]", usuarioID)
	}
	return u.Nombre()
}

// ---------------------------------------------------------------------
// Menús
// ---------------------------------------------------------------------

func (a *App) mostrarBanner() {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════╗")
	fmt.Println("║          LeeLibre - Biblioteca Digital     ║")
	fmt.Println("║        Gestión de libros electrónicos      ║")
	fmt.Println("╚════════════════════════════════════════════╝")
}

func (a *App) menuInicio() {
	for {
		fmt.Println("\n--- Menú principal ---")
		fmt.Println("  1. Iniciar sesión")
		fmt.Println("  2. Salir")
		opcion := a.leerLinea("Opción: ")

		switch opcion {
		case "1":
			if err := a.login(); err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			a.menuUsuario()
		case "2":
			fmt.Println("Hasta pronto.")
			return
		default:
			fmt.Println("Opción no válida.")
		}
	}
}

func (a *App) login() error {
	correo := a.leerLinea("Correo: ")
	password := a.leerLinea("Contraseña: ")
	u, err := a.autenticador.Autenticar(correo, password)
	if err != nil {
		return err
	}
	a.usuarioActual = u
	fmt.Printf("\nBienvenido/a, %s (%s)\n", u.Nombre(), u.Rol())
	return nil
}

func (a *App) menuUsuario() {
	if a.usuarioActual.EsAdministrador() {
		a.menuAdministrador()
	} else {
		a.menuLector()
	}
	a.usuarioActual = nil
}

func (a *App) menuLector() {
	for {
		fmt.Println("\n--- Menú de lector ---")
		fmt.Println("  1. Ver catálogo")
		fmt.Println("  2. Buscar libro")
		fmt.Println("  3. Registrar préstamo")
		fmt.Println("  4. Devolver libro")
		fmt.Println("  5. Ver mi historial")
		fmt.Println("  6. Ver mis recomendaciones")
		fmt.Println("  7. Ver mis reservas en cola")
		fmt.Println("  8. Cerrar sesión")
		opcion := a.leerLinea("Opción: ")

		switch opcion {
		case "1":
			a.verCatalogo()
		case "2":
			a.buscarLibro()
		case "3":
			a.registrarPrestamo()
		case "4":
			a.devolverLibro()
		case "5":
			a.verHistorial()
		case "6":
			a.verRecomendaciones()
		case "7":
			a.verMisReservas()
		case "8":
			fmt.Println("Sesión cerrada.")
			return
		default:
			fmt.Println("Opción no válida.")
		}
	}
}

func (a *App) menuAdministrador() {
	for {
		fmt.Println("\n--- Menú de administrador ---")
		fmt.Println("  1. Ver catálogo")
		fmt.Println("  2. Agregar libro")
		fmt.Println("  3. Editar libro")
		fmt.Println("  4. Eliminar libro")
		fmt.Println("  5. Ver reporte de libros más prestados")
		fmt.Println("  6. Ver todos los préstamos activos")
		fmt.Println("  7. Ver reservas activas")
		fmt.Println("  8. Generar todos los reportes a la vez (concurrente)")
		fmt.Println("  9. Cerrar sesión")
		opcion := a.leerLinea("Opción: ")

		switch opcion {
		case "1":
			a.verCatalogo()
		case "2":
			a.agregarLibro()
		case "3":
			a.editarLibro()
		case "4":
			a.eliminarLibro()
		case "5":
			a.verReporteMasPrestados()
		case "6":
			a.verPrestamosActivos()
		case "7":
			a.verReservasActivas()
		case "8":
			a.verResumenConcurrente()
		case "9":
			fmt.Println("Sesión cerrada.")
			return
		default:
			fmt.Println("Opción no válida.")
		}
	}
}

// ---------------------------------------------------------------------
// Acciones del lector
// ---------------------------------------------------------------------

func (a *App) verCatalogo() {
	libros := a.catalogo.Todos()
	fmt.Printf("\nCatálogo (%d libros):\n", len(libros))
	for _, l := range libros {
		fmt.Printf("  %s\n", l)
	}
}

func (a *App) buscarLibro() {
	texto := a.leerLinea("Buscar por título o autor: ")
	if texto == "" {
		fmt.Println("Debes ingresar un término de búsqueda.")
		return
	}
	resultados := a.catalogo.BuscarPorTexto(texto)
	if len(resultados) == 0 {
		fmt.Println("No se encontraron libros con ese texto.")
		return
	}
	fmt.Printf("\n%d resultado(s):\n", len(resultados))
	for _, l := range resultados {
		fmt.Printf("  %s\n", l)
	}
}

// registrarPrestamo implementa la rama completa del diagrama de flujo:
// si el libro está disponible se presta; si no, se ofrece unirse a la cola
// de reservas (rama "libro no disponible" que faltaba en la versión anterior).
func (a *App) registrarPrestamo() {
	libroID, err := a.leerEntero("ID del libro a prestar: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	libro, err := a.catalogo.BuscarPorID(libroID)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if !libro.Disponible() {
		// Rama del diagrama: libro no disponible - ofrecer cola de reservas
		posActual := a.gestorReservas.PosicionEnCola(libroID, a.usuarioActual.ID())
		if posActual > 0 {
			fmt.Printf("Ya estás en la cola de este libro en la posición #%d.\n", posActual)
			return
		}
		fmt.Printf("El libro \"%s\" no está disponible.\n", libro.Titulo())
		respuesta := a.leerLinea("¿Deseas unirte a la cola de reservas? (s/n): ")
		if strings.ToLower(respuesta) != "s" {
			return
		}
		if err := a.gestorReservas.Reservar(libroID, a.usuarioActual.ID()); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		pos := a.gestorReservas.PosicionEnCola(libroID, a.usuarioActual.ID())
		fmt.Printf("Estás en la cola en la posición #%d.\n", pos)
		return
	}

	// Libro disponible: préstamo directo
	if err := libro.Prestar(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	p, err := a.historial.Registrar(a.usuarioActual.ID(), libroID)
	if err != nil {
		libro.Devolver()
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Préstamo registrado: %s\n", p)
	a.guardarCambios()
}

// devolverLibro devuelve un libro y, si hay reservas en cola, notifica al
// próximo usuario que ya puede prestarlo.
func (a *App) devolverLibro() {
	prestamosMios := a.historial.PorUsuario(a.usuarioActual.ID())
	var activos []*prestamos.Prestamo
	for _, p := range prestamosMios {
		if p.EstaActivo() {
			activos = append(activos, p)
		}
	}
	if len(activos) == 0 {
		fmt.Println("No tienes préstamos activos.")
		return
	}
	fmt.Println("\nTus préstamos activos:")
	for _, p := range activos {
		fmt.Printf("  Préstamo #%d — %s\n", p.ID(), a.tituloLibro(p.LibroID()))
	}
	id, err := a.leerEntero("ID del préstamo a devolver: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	p, err := a.historial.BuscarPorID(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if p.UsuarioID() != a.usuarioActual.ID() {
		fmt.Printf("Error: %v\n", errdom.ErrPermisoDenegado)
		return
	}
	if err := p.Devolver(); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if libro, err := a.catalogo.BuscarPorID(p.LibroID()); err == nil {
		libro.Devolver()
		// Notificar al próximo en la cola de reservas si existe
		siguienteID, err := a.gestorReservas.LiberarLibro(p.LibroID())
		if err == nil && siguienteID > 0 {
			fmt.Printf("Aviso: el libro \"%s\" queda disponible y se notifica a %s (siguiente en cola).\n",
				libro.Titulo(), a.nombreUsuario(siguienteID))
		}
	}
	fmt.Println("Libro devuelto correctamente.")
	a.guardarCambios()
}

// verHistorial usa el helper tituloLibro para evitar el crash cuando un
// libro fue eliminado del catálogo pero sigue referenciado en el historial.
func (a *App) verHistorial() {
	prestamosMios := a.historial.PorUsuario(a.usuarioActual.ID())
	if len(prestamosMios) == 0 {
		fmt.Println("Aún no tienes préstamos registrados.")
		return
	}
	fmt.Printf("\nTu historial (%d préstamos):\n", len(prestamosMios))
	for _, p := range prestamosMios {
		fmt.Printf("  Préstamo #%d — %s — %s\n",
			p.ID(), a.tituloLibro(p.LibroID()), p.Estado())
	}
}

func (a *App) verRecomendaciones() {
	// Polimorfismo: la variable r tiene tipo interface Reportador,
	// pero apunta a un *Recomendador concreto.
	var r reportes.Reportador = reportes.NewRecomendador(
		a.catalogo, a.historial, a.usuarioActual, 5)
	fmt.Printf("\n%s\n", r.Nombre())
	fmt.Println(r.Generar())
}

// verMisReservas muestra los libros en los que el usuario actual está en cola.
func (a *App) verMisReservas() {
	fmt.Println("\nTus reservas en cola:")
	encontradas := 0
	for _, libro := range a.catalogo.Todos() {
		pos := a.gestorReservas.PosicionEnCola(libro.ID(), a.usuarioActual.ID())
		if pos > 0 {
			fmt.Printf("  \"%s\" — posición #%d en la cola\n", libro.Titulo(), pos)
			encontradas++
		}
	}
	if encontradas == 0 {
		fmt.Println("  No tienes reservas activas en ninguna cola.")
	}
}

// ---------------------------------------------------------------------
// Acciones del administrador
// ---------------------------------------------------------------------

func (a *App) agregarLibro() {
	fmt.Println("\n-- Nuevo libro --")
	id, err := a.leerEntero("ID: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	titulo := a.leerLinea("Título: ")
	autor := a.leerLinea("Autor: ")
	fmt.Println("Categorías disponibles:")
	for _, c := range a.catalogo.TodasCategorias() {
		fmt.Printf("  %d. %s\n", c.ID(), c.Nombre())
	}
	catID, err := a.leerEntero("ID de categoría: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	anio, err := a.leerEntero("Año: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	formato := a.leerLinea("Formato (PDF/EPUB): ")

	libro, err := catalogo.NewLibro(id, titulo, autor, catID, anio, catalogo.Formato(formato))
	if err != nil {
		if errors.Is(err, errdom.ErrDatosInvalidos) {
			fmt.Printf("Datos inválidos: %v\n", err)
			return
		}
		fmt.Printf("Error inesperado: %v\n", err)
		return
	}
	if err := a.catalogo.AgregarLibro(libro); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Libro agregado con éxito.")
	a.guardarCambios()
}

// editarLibro modifica un libro ya existente. Cada dato se pide por separado
// y se deja en blanco para conservarlo, de modo que se puede corregir solo el
// año sin volver a teclear el título.
//
// Los cambios pasan por los setters del libro, que son los que aplican las
// reglas del dominio: si el año está fuera de rango, el setter lo rechaza y
// el libro se queda como estaba.
func (a *App) editarLibro() {
	id, err := a.leerEntero("ID del libro a editar: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	libro, err := a.catalogo.BuscarPorID(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("\nEditando: %s\n", libro)
	fmt.Println("(dejar en blanco para no cambiar el dato)")

	if nuevo := a.leerLinea(fmt.Sprintf("Título [%s]: ", libro.Titulo())); nuevo != "" {
		if err := libro.SetTitulo(nuevo); err != nil {
			fmt.Printf("Datos inválidos: %v\n", err)
			return
		}
	}
	if nuevo := a.leerLinea(fmt.Sprintf("Autor [%s]: ", libro.Autor())); nuevo != "" {
		if err := libro.SetAutor(nuevo); err != nil {
			fmt.Printf("Datos inválidos: %v\n", err)
			return
		}
	}
	if nuevo := a.leerLinea(fmt.Sprintf("Año [%d]: ", libro.Anio())); nuevo != "" {
		anio, err := strconv.Atoi(nuevo)
		if err != nil {
			fmt.Println("El año debe ser un número.")
			return
		}
		if err := libro.SetAnio(anio); err != nil {
			fmt.Printf("Datos inválidos: %v\n", err)
			return
		}
	}
	if nuevo := a.leerLinea(fmt.Sprintf("Formato PDF/EPUB [%s]: ", libro.Formato())); nuevo != "" {
		if err := libro.SetFormato(catalogo.Formato(nuevo)); err != nil {
			fmt.Printf("Datos inválidos: %v\n", err)
			return
		}
	}
	fmt.Printf("Libro actualizado: %s\n", libro)
	a.guardarCambios()
}

func (a *App) eliminarLibro() {
	id, err := a.leerEntero("ID del libro a eliminar: ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if err := a.catalogo.EliminarLibro(id); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println("Libro eliminado.")
	a.guardarCambios()
}

// verResumenConcurrente genera varios reportes al mismo tiempo en lugar de
// uno detrás de otro, y muestra cuánto tardó cada uno junto al tiempo total.
//
// La comparación es la parte interesante: la suma de las duraciones
// individuales es mayor que el tiempo total transcurrido, y eso solo puede
// pasar si se ejecutaron solapados.
func (a *App) verResumenConcurrente() {
	lista := []reportes.Reportador{
		reportes.NewMasPrestados(a.catalogo, a.historial, 5),
	}
	for _, u := range a.autenticador.Todos() {
		if len(a.historial.PorUsuario(u.ID())) > 0 {
			lista = append(lista, reportes.NewRecomendador(a.catalogo, a.historial, u, 3))
		}
	}

	fmt.Printf("\nGenerando %d reportes en paralelo...\n", len(lista))

	inicio := time.Now()
	resultados := reportes.EjecutarTodosConcurrente(lista)
	total := time.Since(inicio)

	fmt.Print(reportes.FormatearResultados(resultados))

	var suma time.Duration
	for _, r := range resultados {
		suma += r.Duracion
	}
	fmt.Printf("\nTiempo total transcurrido: %v\n", total)
	fmt.Printf("Suma de los tiempos individuales: %v\n", suma)
	fmt.Println("La suma es mayor que el total porque los reportes se generaron al mismo tiempo.")
}

func (a *App) verReporteMasPrestados() {
	var r reportes.Reportador = reportes.NewMasPrestados(a.catalogo, a.historial, 5)
	fmt.Printf("\n%s\n", r.Nombre())
	fmt.Println(r.Generar())
}

func (a *App) verPrestamosActivos() {
	todos := a.historial.Todos()
	var activos []*prestamos.Prestamo
	for _, p := range todos {
		if p.EstaActivo() {
			activos = append(activos, p)
		}
	}
	if len(activos) == 0 {
		fmt.Println("No hay préstamos activos.")
		return
	}
	fmt.Printf("\nPréstamos activos (%d):\n", len(activos))
	for _, p := range activos {
		fmt.Printf("  #%d — %s — usuario: %s — %s\n",
			p.ID(), a.tituloLibro(p.LibroID()), a.nombreUsuario(p.UsuarioID()), p.Estado())
	}
}

func (a *App) verReservasActivas() {
	total := a.gestorReservas.TotalReservas()
	if total == 0 {
		fmt.Println("No hay reservas activas en el sistema.")
		return
	}
	fmt.Printf("\nReservas activas (%d en total):\n", total)
	for _, libro := range a.catalogo.Todos() {
		cola := a.gestorReservas.ColaDe(libro.ID())
		if cola.Tamano() == 0 {
			continue
		}
		fmt.Printf("  \"%s\": %d en cola\n", libro.Titulo(), cola.Tamano())
		for i, uid := range cola.Elementos() {
			fmt.Printf("    #%d — %s\n", i+1, a.nombreUsuario(uid))
		}
	}
}

// ---------------------------------------------------------------------
// Punto de entrada
// ---------------------------------------------------------------------

func main() {
	app, err := NewApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error al iniciar: %v\n", err)
		os.Exit(1)
	}
	app.mostrarBanner()
	fmt.Println("\nUsuarios de prueba disponibles:")
	fmt.Println("  ronny@leelibre.ec  / ronny123  (administrador)")
	fmt.Println("  ana@leelibre.ec    / ana123    (lectora de ficción)")
	fmt.Println("  carlos@leelibre.ec / carlos123 (lector de ciencia)")
	fmt.Println("  juan@leelibre.ec   / juan123   (lector de tecnología)")
	app.menuInicio()
}
