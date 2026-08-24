package reportes

import (
	"fmt"
	"testing"
	"time"
)

// reporteFalso es una implementación mínima de Reportador para las pruebas.
// Que baste con declarar los dos métodos, sin anunciar en ninguna parte que
// implementa la interface, es justo la ventaja de las interfaces de Go: se
// puede crear un doble de prueba sin tocar el código de producción.
type reporteFalso struct {
	nombre string
	tarda  time.Duration
}

func (r reporteFalso) Nombre() string { return r.nombre }

func (r reporteFalso) Generar() string {
	time.Sleep(r.tarda) // simula un cálculo costoso
	return "contenido de " + r.nombre
}

// TestEjecutarTodosConcurrenteDevuelveTodo comprueba que no se pierde ningún
// reporte por el camino: si el canal o el WaitGroup estuvieran mal, faltarían
// resultados o el programa se quedaría colgado.
func TestEjecutarTodosConcurrenteDevuelveTodo(t *testing.T) {
	entrada := []Reportador{
		reporteFalso{"primero", 10 * time.Millisecond},
		reporteFalso{"segundo", 10 * time.Millisecond},
		reporteFalso{"tercero", 10 * time.Millisecond},
	}

	resultados := EjecutarTodosConcurrente(entrada)

	if len(resultados) != len(entrada) {
		t.Fatalf("se esperaban %d resultados, llegaron %d", len(entrada), len(resultados))
	}
	for i, r := range resultados {
		esperado := entrada[i].Nombre()
		if r.Nombre != esperado {
			t.Errorf("posición %d: se esperaba %q, llegó %q", i, esperado, r.Nombre)
		}
		if r.Contenido == "" {
			t.Errorf("el reporte %q llegó sin contenido", r.Nombre)
		}
	}
}

// TestOrdenEsSiempreElMismo es la prueba que justifica el sort del final. Las
// goroutines terminan en orden impredecible: sin reordenar, la salida
// cambiaría entre ejecuciones. Se repite varias veces porque un fallo de
// concurrencia puede no aparecer en el primer intento.
func TestOrdenEsSiempreElMismo(t *testing.T) {
	entrada := []Reportador{
		reporteFalso{"lento", 30 * time.Millisecond},
		reporteFalso{"rápido", 1 * time.Millisecond},
		reporteFalso{"medio", 15 * time.Millisecond},
	}

	for intento := 0; intento < 5; intento++ {
		resultados := EjecutarTodosConcurrente(entrada)
		if resultados[0].Nombre != "lento" || resultados[1].Nombre != "rápido" || resultados[2].Nombre != "medio" {
			t.Fatalf("intento %d: el orden no se respetó: %v, %v, %v",
				intento, resultados[0].Nombre, resultados[1].Nombre, resultados[2].Nombre)
		}
	}
}

// TestConcurrenteEsMasRapidoQueSecuencial comprueba que la concurrencia
// realmente sirve para algo aquí, no que solo compila.
//
// Tres reportes de 60 ms cada uno tardarían unos 180 ms en secuencia. En
// paralelo deben acercarse a 60. Se compara contra 150 ms y no contra 60
// exactos para que la prueba no falle en un equipo cargado: el margen es
// amplio a propósito, porque una prueba que falla a veces sin motivo real
// acaba ignorándose.
func TestConcurrenteEsMasRapidoQueSecuencial(t *testing.T) {
	const cuantos = 3
	const cadaUno = 60 * time.Millisecond

	entrada := make([]Reportador, 0, cuantos)
	for i := 0; i < cuantos; i++ {
		entrada = append(entrada, reporteFalso{fmt.Sprintf("reporte-%d", i), cadaUno})
	}

	inicio := time.Now()
	EjecutarTodosConcurrente(entrada)
	transcurrido := time.Since(inicio)

	secuencial := cadaUno * cuantos // 180 ms
	limite := 150 * time.Millisecond

	if transcurrido > limite {
		t.Errorf("tardó %v; en secuencia serían ~%v, así que no se están ejecutando en paralelo",
			transcurrido, secuencial)
	}
	t.Logf("%d reportes de %v cada uno: %v en paralelo frente a ~%v en secuencia",
		cuantos, cadaUno, transcurrido.Round(time.Millisecond), secuencial)
}

// TestListaVaciaNoSeCuelga cubre el caso límite: sin reportes que generar, la
// función debe devolver una lista vacía y no quedarse esperando en el Wait.
func TestListaVaciaNoSeCuelga(t *testing.T) {
	resultados := EjecutarTodosConcurrente([]Reportador{})
	if len(resultados) != 0 {
		t.Errorf("se esperaba una lista vacía, llegaron %d resultados", len(resultados))
	}
}
