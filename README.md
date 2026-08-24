# LeeLibre

Sistema de gestión de biblioteca digital desarrollado en **Go**, aplicando programación orientada a objetos, concurrencia y servicios web REST.

**Proyecto integrador** — Programación Orientada a Objetos 2
**Autor:** Cortez Villa Ronny (trabajo individual)
**Docente:** Ing. Torres Arévalo Carlos
**Universidad Internacional del Ecuador**
**Fecha de entrega:** 23 de agosto de 2026

---

## Objetivo del programa

Administrar una biblioteca digital: mantener un catálogo de libros, controlar quién los tiene prestados y hasta cuándo, gestionar la espera cuando un título no está disponible, y ofrecer al lector recomendaciones basadas en lo que ya ha leído.

El sistema se usa de dos formas sobre el mismo núcleo: una **aplicación de consola** para operar localmente y un **servidor de servicios web REST** que expone las mismas operaciones en JSON.

---

## Funcionalidades principales

**Catálogo**
- Consultar todos los libros, buscar por texto (título o autor) y filtrar por categoría o disponibilidad.
- Agregar, modificar y retirar libros. Reservado al administrador.
- Validación de datos: no se admite un título vacío, un año fuera del rango 1000–2100 ni un formato distinto de PDF o EPUB.

**Usuarios y autenticación**
- Dos roles: lector y administrador, con permisos distintos.
- La contraseña no se puede leer, solo verificar.
- En el API, autenticación básica de HTTP mediante un middleware.

**Préstamos**
- Préstamo por 14 días con cálculo automático del vencimiento.
- Tres estados: vigente, devuelto y vencido.
- Historial por usuario y por libro.
- Un préstamo solo lo puede devolver su titular o un administrador.

**Reservas**
- Si el libro está prestado, el usuario entra en una **cola FIFO** en lugar de recibir solo un error.
- Al devolverse el libro, el primero de la fila queda notificado.
- Consulta de la posición en la cola.

**Reportes**
- Libros más prestados.
- **Recomendador personalizado**: construye el perfil del lector a partir de su historial y puntúa los libros que aún no ha leído, indicando el motivo de cada sugerencia.
- Generación de varios reportes **en paralelo**.

---

## Cómo ejecutar

Requiere Go 1.22 o superior. **No hay dependencias externas que instalar.**

```bash
# Aplicación de consola
go run ./cmd/leelibre

# Servidor de servicios web (puerto 8080 por defecto)
go run ./cmd/server
PORT=3000 go run ./cmd/server   # con otro puerto

# Pruebas
go test ./...                   # toda la batería
go test -v ./internal/reservas  # detalle de un paquete
go test -race ./...             # con detector de condiciones de carrera
go test -cover ./...            # con porcentaje de cobertura
```

---

## Servicios web

Trece rutas, todas con respuesta en JSON.

| Método | Ruta | Descripción | Acceso |
|---|---|---|---|
| `GET` | `/` | Información del API y sus rutas | Público |
| `GET` | `/api/libros` | Listar catálogo (`?texto=` y `?categoria=`) | Público |
| `GET` | `/api/libros/{id}` | Detalle de un libro | Público |
| `POST` | `/api/libros` | Crear libro | Administrador |
| `PUT` | `/api/libros/{id}` | Modificar libro | Administrador |
| `DELETE` | `/api/libros/{id}` | Eliminar libro | Administrador |
| `GET` | `/api/categorias` | Listar categorías | Público |
| `POST` | `/api/prestamos` | Registrar préstamo, o encolar si no está disponible | Autenticado |
| `POST` | `/api/prestamos/{id}/devolver` | Devolver un libro | Titular o administrador |
| `GET` | `/api/usuarios/{id}/recomendaciones` | Recomendaciones personalizadas | Público |
| `GET` | `/api/reportes/mas-prestados` | Top de libros más prestados | Público |
| `GET` | `/api/reportes/resumen` | Varios reportes generados **en paralelo** | Público |
| `GET` | `/api/reservas/{libro_id}` | Cola de espera de un libro | Público |

### Autenticación

Los endpoints protegidos usan autenticación básica de HTTP, resuelta con un middleware sobre `r.BasicAuth()`. En `curl` basta con `-u correo:contraseña`.

- Sin credenciales o incorrectas → **401**
- Autenticado pero sin permiso → **403**

El préstamo se registra **siempre a nombre de quien se autentica**; el cuerpo solo lleva `libro_id`. Si el cliente pudiera enviar un `usuario_id`, cualquiera podría pedir libros en nombre de otra persona.

### Ejemplos

```bash
# Buscar libros
curl "http://localhost:8080/api/libros?texto=cosmos"

# Recomendaciones para Ana (usuario 2)
curl http://localhost:8080/api/usuarios/2/recomendaciones

# Varios reportes a la vez: la respuesta trae el tiempo de cada uno
curl "http://localhost:8080/api/reportes/resumen?usuario=2"

# Registrar un préstamo (a nombre de quien se autentica)
curl -X POST http://localhost:8080/api/prestamos \
  -u ana@leelibre.ec:ana123 \
  -H "Content-Type: application/json" \
  -d '{"libro_id": 5}'

# Modificar un libro (solo administrador)
curl -X PUT http://localhost:8080/api/libros/1 \
  -u ronny@leelibre.ec:ronny123 \
  -H "Content-Type: application/json" \
  -d '{"anio": 1970}'
```

### Usuarios de prueba

| Correo | Contraseña | Rol | Perfil de lectura |
|---|---|---|---|
| `ronny@leelibre.ec` | `ronny123` | Administrador | — |
| `ana@leelibre.ec` | `ana123` | Lectora | Ficción |
| `carlos@leelibre.ec` | `carlos123` | Lector | Ciencia |
| `maria@leelibre.ec` | `maria123` | Lectora | Historia y filosofía |
| `juan@leelibre.ec` | `juan123` | Lector | Tecnología |

---

## Mapeo de los contenidos de la asignatura

Dónde queda implementado cada tema de las cuatro unidades.

### Unidad 1 — Fundamentos (semanas 1 y 2)

| Tema | Dónde | Ejemplo |
|---|---|---|
| Sintaxis, variables, condicionales | Todo el proyecto | `internal/catalogo/libro.go` |
| Funciones y valores de retorno múltiples | Todo el proyecto | `NewLibro() (*Libro, error)` |
| Paquetes e imports locales | 8 paquetes en `internal/` | `cmd/server/main.go` |
| Bucles | Recorridos y menús | `menuLector()` en `cmd/leelibre/main.go` |

### Unidad 2 — Estructuras de datos (semanas 3 y 4)

| Tema | Dónde | Por qué |
|---|---|---|
| **Slice** | Catálogo de libros, historial de préstamos, cola de reservas | La colección crece y se recorre entera con frecuencia |
| **Map** | Índices por id y por correo, perfil del lector, conteo por libro | Acceso directo por clave sin recorrer la colección |
| **Array** | *No se usa* | Ninguna colección del dominio tiene tamaño fijo conocido de antemano; forzar un array daría un límite artificial de libros o usuarios |
| **Struct** | `Libro`, `Categoria`, `Usuario`, `Prestamo`, `Cola`, `Catalogo`… | Cada entidad del dominio |
| **Métodos** | Todas las entidades | `libro.Prestar()`, `prestamo.Devolver()` |
| **Constructores** | Función `New*` por tipo | `NewLibro`, `NewUsuario`, `NewPrestamo` |
| **Cola FIFO** | `internal/reservas/reservas.go` | El orden de llegada decide quién recibe el libro primero |

### Unidad 3 — Programación orientada a objetos (semanas 5 y 6)

| Tema | Dónde | Ejemplo |
|---|---|---|
| **Encapsulación** | Todos los campos en minúscula | `Usuario.password` no tiene captador: solo se verifica |
| **Captadores idiomáticos** | Sin prefijo `Get` | `libro.Titulo()`, no `libro.GetTitulo()` |
| **Modificadores con validación** | `internal/catalogo/libro.go` | `SetAnio` rechaza años fuera de 1000–2100 |
| **Receptores de puntero y de valor** | Puntero cuando se modifica | `func (l *Libro) Prestar() error` |
| **Manejo de errores** | `internal/errores/errores.go` | Valores centinela comparados con `errors.Is` |
| **Envoltura de errores** | Todo el dominio | `fmt.Errorf("%w: …", errores.ErrDatosInvalidos)` |
| **Interfaces y polimorfismo** | `internal/reportes/reportador.go` | `Reportador`, con dos implementaciones |
| **`fmt.Stringer`** | `Libro.String()`, `Prestamo.String()` | Interface de la biblioteca estándar |

### Unidad 4 — Concurrencia, servicios web y pruebas (semanas 7 y 8)

| Tema | Dónde | Para qué |
|---|---|---|
| **Goroutines** | `internal/reportes/concurrente.go`, `cmd/server/main.go` | Generar reportes a la vez; cargar los archivos de datos en paralelo |
| **`sync.WaitGroup`** | Los mismos dos sitios | Esperar a que terminen las goroutines antes de continuar |
| **Canales** | `internal/reportes/concurrente.go` | Cada goroutine entrega su resultado por el canal |
| **Canal con búfer** | `EjecutarTodosConcurrente` | Evita que las goroutines se bloqueen al escribir |
| **`sync.RWMutex`** | `Catalogo`, `Historial`, `Cola`, `Libro`, `Prestamo` | El servidor atiende cada petición en su goroutine: sin candado, dos peticiones corromperían el estado |
| **`sync/atomic`** | `cmd/server/main.go` | Contador de peticiones atendidas, sin bloqueo |
| **Condiciones de carrera** | Verificado con `go test -race` | 12 accesos conflictivos detectados y corregidos |
| **Servicios web REST** | `internal/api/server.go` | 13 rutas con `net/http` |
| **Serialización JSON** | DTOs en `internal/api/server.go` | `encoding/json` con etiquetas de campo |
| **Patrón middleware** | `conAuth`, `conAdmin`, `loggingMiddleware` | Autenticación y registro sin repetirlos en cada handler |
| **Pruebas unitarias** | 6 archivos `_test.go` | 48 funciones de prueba |
| **Pruebas basadas en tabla** | `libro_test.go`, `prestamo_test.go` | Un caso por fila, con `t.Run` |
| **Pruebas de HTTP** | `internal/api/server_test.go` | `httptest.NewRecorder` sin abrir puertos |

---

## Estructura del proyecto

```
leelibre/
├── cmd/
│   ├── leelibre/main.go          Aplicación de consola
│   └── server/main.go            Servidor REST + carga concurrente
├── internal/
│   ├── errores/                  Errores centinela del dominio
│   ├── catalogo/                 Libros, categorías y catálogo
│   ├── usuarios/                 Usuarios, roles y autenticación
│   ├── prestamos/                Préstamos e historial
│   ├── reservas/                 Cola FIFO de espera
│   ├── reportes/                 Interface Reportador y ejecución concurrente
│   ├── persistencia/             Guardado en JSON con escritura atómica
│   └── api/                      Handlers HTTP, DTOs y middleware
├── data/                         Datos de ejemplo (7 categorías, 30 libros, 10 usuarios)
├── docs/                         Documentación y diagramas de las tres etapas
├── go.mod                        Sin dependencias externas
└── README.md
```

La dirección de las dependencias es un criterio sostenido: los paquetes del dominio no conocen la capa web, y `catalogo` no depende de `usuarios` ni de `prestamos`. Por eso los dos ejecutables comparten el mismo núcleo sin duplicar lógica.

---

## Pruebas

```
48 funciones de prueba en 6 paquetes, todas en verde con el detector de carreras.

paquete                cobertura
internal/reservas          94.1 %
internal/catalogo          65.8 %
internal/api               60.4 %
internal/prestamos         54.7 %
internal/usuarios          51.0 %
```

Lo que cubren, más allá del porcentaje:

- **Reglas del dominio**: que un libro prestado no se pueda prestar dos veces, que un préstamo venza a los 14 días, que devolver dos veces falle.
- **Validaciones**: que los modificadores rechacen lo imposible **y que el objeto conserve su valor anterior**, no que quede a medio cambiar.
- **Seguridad**: la matriz completa de permisos (401 / 403 / 200) y que un usuario no pueda pedir un libro a nombre de otro ni devolver un préstamo ajeno.
- **Encapsulación**: que los métodos que devuelven colecciones entreguen una copia, de modo que quien la reciba no pueda alterar el estado interno.
- **Concurrencia**: 50 reservas simultáneas sin perder ninguna, y que los reportes en paralelo tarden menos que en secuencia (61 ms frente a 180 ms).

Para poder probar el vencimiento sin esperar catorce días ni cambiar la hora del computador, `ActualizarEstadoEn(momento)` recibe la fecha como parámetro en lugar de leer el reloj del sistema.

---

## Decisiones de diseño

**Sin dependencias externas.** La planeación inicial contemplaba Gin, SQLite, JWT y bcrypt; no se usó ninguno. Desde Go 1.22 el enrutador de la biblioteca estándar admite métodos y variables de ruta, que era lo que aportaba el framework. El proyecto compila sin descargar nada y las decisiones de diseño quedan a la vista en el código, no delegadas.

**Persistencia en archivos JSON.** Para este volumen la diferencia con una base de datos es irrelevante, y la capa quedó aislada: migrarla no afectaría al resto de módulos. La escritura es atómica —primero a un archivo temporal y luego renombrado— para que una interrupción no deje el archivo corrupto.

**Del paradigma funcional al orientado a objetos.** La planeación de la Etapa 1 proponía programación funcional. Se corrigió porque los contenidos de la asignatura —estructuras, métodos, encapsulación, interfaces— pertenecen al paradigma orientado a objetos. Lo que sí encajaba se conservó: `Catalogo.Filtrar` recibe una función como parámetro, y las búsquedas se construyen sobre ella en lugar de repetir el recorrido.

---

## Alcance y limitaciones

**Incluido:** catálogo completo, usuarios con roles, préstamos con estados, cola de reservas, recomendador, 13 servicios REST con autenticación, persistencia en JSON, concurrencia y batería de pruebas.

**No incluido, y por qué:**

- **Interfaz web visual.** El proyecto expone JSON; consumirlo desde un navegador corresponde a programación web y queda fuera del alcance de esta asignatura.
- **Base de datos.** Ver decisiones de diseño.
- **Contraseñas cifradas.** Se guardan en texto plano en los datos de ejemplo, que son ficticios. En un sistema real irían con `bcrypt`; se documenta como limitación consciente y no como olvido.
- **Cobertura completa de pruebas.** Se priorizaron las reglas de negocio y la seguridad sobre el porcentaje. Los paquetes `persistencia` y los `cmd` no tienen pruebas propias.

---

## Documentación

En `docs/`:
- Documento de la Etapa 1 — planeación
- Documento de la Etapa 2 — desarrollo
- Documento final — integración de las cuatro unidades
- `diagrams/` — diagramas editables en formato draw.io, incluido el diagrama de clases del sistema

---

## Licencia

MIT — ver el archivo `LICENSE`.
