# LeeLibre

Sistema de gestión de biblioteca digital desarrollado en Go bajo el paradigma de **Programación Orientada a Objetos**. Proyecto integrador de la asignatura *Programación Orientada a Objetos 2*.

Estado: **Etapa 2 completada** — dominio de POO, cola de reservas, persistencia en JSON y servicios web REST.

## Descripción

LeeLibre administra una biblioteca digital con dos interfaces sobre el mismo núcleo:

- **CLI (consola)** — herramienta de operación local para administradores y lectores.
- **API REST** — servicios web con serialización JSON, listos para consumo desde cualquier cliente HTTP.

Ambas comparten los mismos paquetes del dominio (`catalogo`, `usuarios`, `prestamos`, `reservas`, `reportes`), demostrando que la arquitectura por capas permite exponer la misma lógica a través de múltiples interfaces sin duplicación.

## Stack tecnológico

- **Lenguaje:** Go 1.22+
- **Servidor web:** `net/http` (biblioteca estándar — sin dependencias externas)
- **Serialización:** JSON con `encoding/json`
- **Persistencia:** archivos JSON en disco
- **Cero dependencias externas** — el proyecto compila con solo la biblioteca estándar

## Estructura del proyecto

```
leelibre/
├── cmd/
│   ├── leelibre/main.go          # Aplicación CLI
│   └── server/main.go            # Servidor HTTP REST
├── internal/
│   ├── errores/errores.go        # Errores de dominio
│   ├── catalogo/                 # Libros, categorías, catálogo
│   ├── usuarios/                 # Usuarios, autenticación, roles
│   ├── prestamos/                # Préstamos e historial
│   ├── reservas/                 # ★ Cola FIFO de reservas (estructura de datos)
│   ├── reportes/                 # Interface Reportador + implementaciones
│   ├── persistencia/             # Guardado a JSON de cambios
│   └── api/                      # Handlers HTTP y DTOs para JSON
├── data/
│   ├── categorias.json           # 7 categorías
│   ├── libros.json               # 30 libros
│   ├── usuarios.json             # 10 usuarios
│   └── prestamos.json            # Se genera al ejecutar (persistencia)
├── docs/                         # Documentación de etapas
├── go.mod                        # Sin dependencias
└── README.md
```

## Cómo ejecutar

### Modo CLI (menú por consola)

```bash
go run ./cmd/leelibre
```

### Modo servidor REST (servicios web)

```bash
go run ./cmd/server
```

Por defecto escucha en el puerto 8080. Se puede cambiar con la variable de entorno `PORT`:

```bash
PORT=3000 go run ./cmd/server
```

## Endpoints REST disponibles

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/` | Información del API |
| `GET` | `/api/libros` | Listar catálogo (soporta `?texto=` y `?categoria=`) |
| `GET` | `/api/libros/{id}` | Detalle de un libro |
| `POST` | `/api/libros` | Crear libro |
| `DELETE` | `/api/libros/{id}` | Eliminar libro |
| `GET` | `/api/categorias` | Listar categorías |
| `POST` | `/api/prestamos` | Registrar préstamo (o encolar si no disponible) |
| `POST` | `/api/prestamos/{id}/devolver` | Devolver un libro |
| `GET` | `/api/usuarios/{id}/recomendaciones` | Recomendaciones personalizadas |
| `GET` | `/api/reportes/mas-prestados` | Top de libros más prestados |
| `GET` | `/api/reservas/{libro_id}` | Ver cola de espera de un libro |

### Ejemplos de uso

```bash
# Listar todos los libros
curl http://localhost:8080/api/libros

# Buscar libros por texto
curl "http://localhost:8080/api/libros?texto=cosmos"

# Ver recomendaciones para Ana (usuario 2)
curl http://localhost:8080/api/usuarios/2/recomendaciones

# Registrar un préstamo
curl -X POST http://localhost:8080/api/prestamos \
  -H "Content-Type: application/json" \
  -d '{"usuario_id": 2, "libro_id": 5}'

# Devolver un préstamo
curl -X POST http://localhost:8080/api/prestamos/1/devolver
```

## Usuarios de prueba

| Correo | Contraseña | Rol | Perfil |
|---|---|---|---|
| `ronny@leelibre.ec` | `ronny123` | Administrador | — |
| `ana@leelibre.ec` | `ana123` | Lector | Ficción |
| `carlos@leelibre.ec` | `carlos123` | Lector | Ciencia |
| `maria@leelibre.ec` | `maria123` | Lector | Historia/Filosofía |
| `juan@leelibre.ec` | `juan123` | Lector | Tecnología |

## Conceptos de POO aplicados

Este proyecto implementa los temas de las **4 unidades** de la asignatura:

- **Unidad 1** — Sintaxis, condicionales, funciones, imports: todo el código
- **Unidad 2** — Arrays, slices, maps, structs, métodos, constructores: todos los paquetes
- **Unidad 3** — Encapsulación por paquete, getters idiomáticos, manejo de errores con `errors.Is`, interfaces con polimorfismo
- **Unidad 4** — Servicios web REST con `net/http`, serialización JSON con `encoding/json`, patrón middleware

### Estructuras de datos

- **Slice** — catálogo de libros, historial de préstamos
- **Map** — índices por id, perfil de lector del recomendador
- **Cola FIFO** — cola de reservas cuando un libro no está disponible (paquete `reservas`)

### Polimorfismo

La interface `Reportador` en `internal/reportes/reportador.go` es implementada por:
- `MasPrestados` (`internal/reportes/mas_prestados.go`)
- `Recomendador` (`internal/reportes/recomendador.go`)

Ambos se usan de forma uniforme desde CLI y API sin conocer el tipo concreto.

## Funcionalidad no básica

El **recomendador personalizado** (`internal/reportes/recomendador.go`) analiza el historial del usuario, construye su perfil de lector (map de categorías), y sugiere libros no leídos con un score de coincidencia. Es la funcionalidad estrella que va más allá del CRUD básico.

## Trazabilidad con la Etapa 1

| Planificado en Etapa 1 | Implementado en Etapa 2 |
|---|---|
| Arquitectura por capas | ✅ Paquetes internos + `api/` como capa web |
| Servicios web con JSON | ✅ `net/http` + 10 endpoints REST |
| Módulo de catálogo | ✅ `internal/catalogo/` |
| Módulo de usuarios | ✅ `internal/usuarios/` con roles |
| Módulo de préstamos | ✅ `internal/prestamos/` con estados |
| Módulo de reportes | ✅ `internal/reportes/` con interface |
| Manejo de errores | ✅ `internal/errores/` con `errors.Is` |
| Rama "libro no disponible" del diagrama de flujo | ✅ `internal/reservas/` (cola FIFO) |

## Autor

**Cortez Villa Ronny** — Estudiante de tercer semestre
Universidad Internacional del Ecuador

## Licencia

MIT — ver archivo `LICENSE`.
