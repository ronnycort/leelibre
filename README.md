# LeeLibre

Sistema de gestión de libros electrónicos desarrollado en Go, siguiendo el paradigma de programación funcional. Proyecto integrador de la asignatura, dividido en tres etapas: planeación, desarrollo e integración.

## Descripción

LeeLibre es una aplicación web para administrar una biblioteca digital. Permite consultar un catálogo de libros, registrar préstamos digitales por parte de usuarios autenticados y consultar el historial de lectura. Un módulo administrativo permite mantener el catálogo actualizado.

## Stack tecnológico

- **Lenguaje:** Go 1.22+
- **Framework web:** Gin
- **Base de datos:** SQLite
- **Autenticación:** JWT con contraseñas cifradas mediante bcrypt

## Módulos

1. **Catálogo** — gestión de libros, autores y categorías.
2. **Usuarios y autenticación** — registro, login y roles.
3. **Préstamos digitales** — reserva, descarga e historial.
4. **Reportes** (transversal) — estadísticas agregadas de uso.

## Estructura del proyecto

leelibre/
├── cmd/server/         # punto de entrada de la aplicación
├── internal/
│   ├── catalogo/       # módulo de catálogo
│   ├── usuarios/       # módulo de usuarios
│   ├── prestamos/      # módulo de préstamos
│   └── reportes/       # módulo transversal
├── web/
│   ├── templates/      # vistas html/template
│   └── static/         # css e imágenes
├── migrations/         # esquemas SQL
└── docs/               # documentación de las tres entregas

## Dependencias planeadas

| Paquete       | Origen  | Uso            |
|---------------|---------|----------------|
| gin-gonic/gin | Tercero | Framework HTTP |
| mattn/go-sqlite3 | Tercero | Driver de SQLite |
| golang-jwt/jwt | Tercero | Emisión de tokens de sesión |
| golang.org/x/crypto/bcrypt | Oficial | Cifrado de contraseñas |

## Diagramas

Los diagramas del sistema en formato editable draw.io están en `docs/diagrams/`. Se pueden abrir y modificar desde [app.diagrams.net](https://app.diagrams.net).

## Estado del proyecto

Este repositorio se encuentra en la **Etapa 1: Planeación del software**. Todavía no contiene código ejecutable. Las carpetas de módulos (`internal/catalogo`, `internal/usuarios`, `internal/prestamos`, `internal/reportes`) están creadas y a la espera de la implementación que se realizará durante la Etapa 2.

El documento con la planeación completa (alcance, arquitectura, diagramas, paquetes y cronograma) se encuentra en `docs/etapa1_planeacion.pdf`.

## Autor

Cortez Villa Ronny.

## Licencia

MIT — ver archivo `LICENSE`.
