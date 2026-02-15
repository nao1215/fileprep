# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[English](../../README.md) | [日本語](../ja/README.md) | [Français](../fr/README.md) | [한국어](../ko/README.md) | [Русский](../ru/README.md) | [中文](../zh-cn/README.md)

![fileprep-logo](../images/fileprep-logo-small.png)

**fileprep** es una biblioteca de Go para limpiar, normalizar y validar datos estructurados (CSV, TSV, LTSV, JSON, JSONL, Parquet y Excel) mediante reglas ligeras basadas en etiquetas struct, con soporte transparente para flujos gzip, bzip2, xz, zstd, zlib, snappy, s2 y lz4.

## ¿Por qué fileprep?

Desarrollé [nao1215/filesql](https://github.com/nao1215/filesql), que permite ejecutar consultas SQL en archivos como CSV, TSV, LTSV, Parquet y Excel. También creé [nao1215/csv](https://github.com/nao1215/csv) para la validación de archivos CSV.

Mientras estudiaba aprendizaje automático, me di cuenta: "Si extiendo [nao1215/csv](https://github.com/nao1215/csv) para soportar los mismos formatos de archivo que [nao1215/filesql](https://github.com/nao1215/filesql), podría combinarlos para realizar operaciones tipo ETL". Esta idea llevó a la creación de **fileprep**: una biblioteca que conecta el preprocesamiento/validación de datos con las consultas SQL basadas en archivos.

## Características

- Soporte de múltiples formatos: CSV, TSV, LTSV, JSON (.json), JSONL (.jsonl), Parquet, Excel (.xlsx)
- Soporte de compresión: gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst), zlib (.z), snappy (.snappy), s2 (.s2), lz4 (.lz4)
- Vinculación de columnas por nombre: Los campos coinciden automáticamente con nombres de columna en `snake_case`, personalizable con la etiqueta `name`
- Preprocesamiento basado en etiquetas struct (`prep`): trim, lowercase, uppercase, valores por defecto
- Validación basada en etiquetas struct (`validate`): required, omitempty y más
- Opciones del procesador: `WithStrictTagParsing()` para detectar errores de configuración de tags, `WithValidRowsOnly()` para filtrar la salida
- Integración con [filesql](https://github.com/nao1215/filesql): Devuelve `io.Reader` para uso directo con filesql
- Informe detallado de errores: Información de fila y columna para cada error

## Instalación

```bash
go get github.com/nao1215/fileprep
```

## Requisitos

- Versión de Go: 1.24 o posterior
- Sistemas Operativos:
  - Linux
  - macOS
  - Windows


## Inicio Rápido

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// User representa un registro de usuario con preprocesamiento y validación
type User struct {
    Name  string `prep:"trim" validate:"required"`
    Email string `prep:"trim,lowercase"`
    Age   string
}

func main() {
    csvData := `name,email,age
  John Doe  ,JOHN@EXAMPLE.COM,30
Jane Smith,jane@example.com,25
`

    processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
    var users []User

    reader, result, err := processor.Process(strings.NewReader(csvData), &users)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Procesadas %d filas, %d válidas\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("Nombre: %q, Email: %q\n", user.Name, user.Email)
    }

    // reader puede pasarse directamente a filesql
    _ = reader
}
```

Salida:
```
Procesadas 2 filas, 2 válidas
Nombre: "John Doe", Email: "john@example.com"
Nombre: "Jane Smith", Email: "jane@example.com"
```

## Antes de usar fileprep

### JSON/JSONL usa una única columna "data"

Los archivos JSON/JSONL se parsean en una única columna llamada `"data"`. Cada elemento del array (JSON) o línea (JSONL) se convierte en una fila que contiene la cadena JSON sin procesar.

```go
type JSONRecord struct {
    Data string `name:"data" prep:"trim" validate:"required"`
}
```

La salida siempre es JSONL compacto. Si un tag prep rompe la estructura JSON, `Process` devuelve `ErrInvalidJSONAfterPrep`. Si todas las filas terminan vacías, devuelve `ErrEmptyJSONOutput`.

### La coincidencia de columnas distingue mayúsculas y minúsculas

`UserName` se mapea a `user_name` mediante snake_case automático. Encabezados como `User_Name`, `USER_NAME`, `userName` **no** coinciden. Use el tag `name` cuando los encabezados difieran:

```go
type Record struct {
    UserName string                 // coincide solo con "user_name"
    Email    string `name:"EMAIL"`  // coincide exactamente con "EMAIL"
}
```

### Encabezados duplicados: la primera columna gana

Si un archivo tiene `id,id,name`, la primera columna `id` se usa para el enlace. La segunda se ignora.

### Las columnas faltantes se convierten en cadenas vacías

Si no existe una columna para un campo struct, el valor es `""`. Agregue `validate:"required"` para detectar esto en tiempo de parseo.

### Excel: solo se procesa la primera hoja

Los archivos `.xlsx` con múltiples hojas ignorarán silenciosamente todas las hojas después de la primera.

## Ejemplos Avanzados

### Preprocesamiento y Validación de Datos Complejos

Este ejemplo demuestra toda la potencia de fileprep: combinando múltiples preprocesadores y validadores para limpiar y validar datos reales desordenados.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee representa datos de empleados con preprocesamiento y validación completos
type Employee struct {
    // ID: rellenar a 6 dígitos, debe ser numérico
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // Nombre: limpiar espacios, requerido alfabético con espacios
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // Email: normalizar a minúsculas, validar formato
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // Departamento: normalizar mayúsculas, debe ser uno de los valores permitidos
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // Salario: mantener solo dígitos, validar rango
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // Teléfono: extraer dígitos, validar formato E.164 después de agregar código de país
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // Fecha de inicio: validar formato datetime
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // ID de gerente: requerido solo si el departamento no es HR
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // Sitio web: corregir esquema faltante, validar URL
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // Datos CSV desordenados del mundo real
    csvData := `id,name,email,department,salary,phone,start_date,manager_id,website
  42,  John   Doe  ,JOHN.DOE@COMPANY.COM,engineering,"$75,000",555-123-4567,2023-01-15,000001,company.com/john
7,Jane Smith,jane@COMPANY.com,  Sales  ,"$120,000",(555) 987-6543,2022-06-01,000002,WWW.LINKEDIN.COM/in/jane
123,Bob Wilson,bob.wilson@company.com,HR,45000,555.111.2222,2024-03-20,,
99,Alice Brown,alice@company.com,Marketing,$88500,555-444-3333,2023-09-10,000003,https://alice.dev
`

    processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
    var employees []Employee

    _, result, err := processor.Process(strings.NewReader(csvData), &employees)
    if err != nil {
        fmt.Printf("Error fatal: %v\n", err)
        return
    }

    fmt.Printf("=== Resultado del Procesamiento ===\n")
    fmt.Printf("Filas totales: %d, Filas válidas: %d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("Empleado %d:\n", i+1)
        fmt.Printf("  ID:           %s\n", emp.EmployeeID)
        fmt.Printf("  Nombre:       %s\n", emp.FullName)
        fmt.Printf("  Email:        %s\n", emp.Email)
        fmt.Printf("  Departamento: %s\n", emp.Department)
        fmt.Printf("  Salario:      %s\n", emp.Salary)
        fmt.Printf("  Teléfono:     %s\n", emp.Phone)
        fmt.Printf("  Fecha Inicio: %s\n", emp.StartDate)
        fmt.Printf("  ID Gerente:   %s\n", emp.ManagerID)
        fmt.Printf("  Sitio Web:    %s\n\n", emp.Website)
    }
}
```

Salida:
```
=== Resultado del Procesamiento ===
Filas totales: 4, Filas válidas: 4

Empleado 1:
  ID:           000042
  Nombre:       John Doe
  Email:        john.doe@company.com
  Departamento: ENGINEERING
  Salario:      75000
  Teléfono:     +15551234567
  Fecha Inicio: 2023-01-15
  ID Gerente:   000001
  Sitio Web:    https://company.com/john

Empleado 2:
  ID:           000007
  Nombre:       Jane Smith
  Email:        jane@company.com
  Departamento: SALES
  Salario:      120000
  Teléfono:     +15559876543
  Fecha Inicio: 2022-06-01
  ID Gerente:   000002
  Sitio Web:    https://www.linkedin.com/in/jane

Empleado 3:
  ID:           000123
  Nombre:       Bob Wilson
  Email:        bob.wilson@company.com
  Departamento: HR
  Salario:      45000
  Teléfono:     +15551112222
  Fecha Inicio: 2024-03-20
  ID Gerente:   000000
  Sitio Web:

Empleado 4:
  ID:           000099
  Nombre:       Alice Brown
  Email:        alice@company.com
  Departamento: MARKETING
  Salario:      88500
  Teléfono:     +15554443333
  Fecha Inicio: 2023-09-10
  ID Gerente:   000003
  Sitio Web:    https://alice.dev
```


### Informe Detallado de Errores

Cuando la validación falla, fileprep proporciona información precisa del error incluyendo número de fila, nombre de columna y razón específica del fallo de validación.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order representa un pedido con reglas de validación estrictas
type Order struct {
    OrderID    string `name:"order_id" validate:"required,uuid4"`
    CustomerID string `name:"customer_id" validate:"required,numeric"`
    Email      string `validate:"required,email"`
    Amount     string `validate:"required,number,gt=0,lte=10000"`
    Currency   string `validate:"required,len=3,uppercase"`
    Country    string `validate:"required,alpha,len=2"`
    OrderDate  string `name:"order_date" validate:"required,datetime=2006-01-02"`
    ShipDate   string `name:"ship_date" validate:"datetime=2006-01-02,gtfield=OrderDate"`
    IPAddress  string `name:"ip_address" validate:"required,ip_addr"`
    PromoCode  string `name:"promo_code" validate:"alphanumeric"`
    Quantity   string `validate:"required,numeric,gte=1,lte=100"`
    UnitPrice  string `name:"unit_price" validate:"required,number,gt=0"`
    TotalCheck string `name:"total_check" validate:"required,eqfield=Amount"`
}

func main() {
    // CSV con múltiples errores de validación
    csvData := `order_id,customer_id,email,amount,currency,country,order_date,ship_date,ip_address,promo_code,quantity,unit_price,total_check
550e8400-e29b-41d4-a716-446655440000,12345,alice@example.com,500.00,USD,US,2024-01-15,2024-01-20,192.168.1.1,SAVE10,2,250.00,500.00
invalid-uuid,abc,not-an-email,-100,US,USA,2024/01/15,2024-01-10,999.999.999.999,PROMO-CODE-TOO-LONG!!,0,0,999
550e8400-e29b-41d4-a716-446655440001,,bob@test,50000,EURO,J1,not-a-date,,2001:db8::1,VALID20,101,-50,50000
123e4567-e89b-42d3-a456-426614174000,99999,charlie@company.com,1500.50,JPY,JP,2024-02-28,2024-02-25,10.0.0.1,VIP,5,300.10,1500.50
`

    processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
    var orders []Order

    _, result, err := processor.Process(strings.NewReader(csvData), &orders)
    if err != nil {
        fmt.Printf("Error fatal: %v\n", err)
        return
    }

    fmt.Printf("=== Informe de Validación ===\n")
    fmt.Printf("Filas totales:   %d\n", result.RowCount)
    fmt.Printf("Filas válidas:   %d\n", result.ValidRowCount)
    fmt.Printf("Filas inválidas: %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("Errores totales: %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== Detalles de Errores ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("Fila %d, Columna '%s': %s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

Salida:
```
=== Informe de Validación ===
Filas totales:   4
Filas válidas:   1
Filas inválidas: 3
Errores totales: 23

=== Detalles de Errores ===
Fila 2, Columna 'order_id': value must be a valid UUID version 4
Fila 2, Columna 'customer_id': value must be numeric
Fila 2, Columna 'email': value must be a valid email address
Fila 2, Columna 'amount': value must be greater than 0
Fila 2, Columna 'currency': value must have exactly 3 characters
Fila 2, Columna 'country': value must have exactly 2 characters
Fila 2, Columna 'order_date': value must be a valid datetime in format: 2006-01-02
Fila 2, Columna 'ip_address': value must be a valid IP address
Fila 2, Columna 'promo_code': value must contain only alphanumeric characters
Fila 2, Columna 'quantity': value must be greater than or equal to 1
Fila 2, Columna 'unit_price': value must be greater than 0
Fila 2, Columna 'ship_date': value must be greater than field OrderDate
Fila 2, Columna 'total_check': value must equal field Amount
Fila 3, Columna 'customer_id': value is required
Fila 3, Columna 'email': value must be a valid email address
Fila 3, Columna 'amount': value must be less than or equal to 10000
Fila 3, Columna 'currency': value must have exactly 3 characters
Fila 3, Columna 'country': value must contain only alphabetic characters
Fila 3, Columna 'order_date': value must be a valid datetime in format: 2006-01-02
Fila 3, Columna 'quantity': value must be less than or equal to 100
Fila 3, Columna 'unit_price': value must be greater than 0
Fila 3, Columna 'ship_date': value must be greater than field OrderDate
Fila 4, Columna 'ship_date': value must be greater than field OrderDate
```

## Etiquetas de Preprocesamiento (`prep`)

Se pueden combinar múltiples etiquetas: `prep:"trim,lowercase,default=N/A"`

### Preprocesadores Básicos

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `trim` | Eliminar espacios al inicio/final | `prep:"trim"` |
| `ltrim` | Eliminar espacios al inicio | `prep:"ltrim"` |
| `rtrim` | Eliminar espacios al final | `prep:"rtrim"` |
| `lowercase` | Convertir a minúsculas | `prep:"lowercase"` |
| `uppercase` | Convertir a mayúsculas | `prep:"uppercase"` |
| `default=value` | Establecer valor por defecto si está vacío | `prep:"default=N/A"` |

### Transformación de Cadenas

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `replace=old:new` | Reemplazar todas las ocurrencias | `prep:"replace=;:,"` |
| `prefix=value` | Agregar cadena al inicio | `prep:"prefix=ID_"` |
| `suffix=value` | Agregar cadena al final | `prep:"suffix=_END"` |
| `truncate=N` | Limitar a N caracteres | `prep:"truncate=100"` |
| `strip_html` | Eliminar etiquetas HTML | `prep:"strip_html"` |
| `strip_newline` | Eliminar saltos de línea (LF, CRLF, CR) | `prep:"strip_newline"` |
| `collapse_space` | Colapsar múltiples espacios en uno | `prep:"collapse_space"` |

### Filtrado de Caracteres

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `remove_digits` | Eliminar todos los dígitos | `prep:"remove_digits"` |
| `remove_alpha` | Eliminar todos los caracteres alfabéticos | `prep:"remove_alpha"` |
| `keep_digits` | Mantener solo dígitos | `prep:"keep_digits"` |
| `keep_alpha` | Mantener solo caracteres alfabéticos | `prep:"keep_alpha"` |
| `trim_set=chars` | Eliminar caracteres especificados de ambos extremos | `prep:"trim_set=@#$"` |

### Relleno

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `pad_left=N:char` | Rellenar a la izquierda hasta N caracteres | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | Rellenar a la derecha hasta N caracteres | `prep:"pad_right=10: "` |

### Preprocesadores Avanzados

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `normalize_unicode` | Normalizar Unicode a formato NFC | `prep:"normalize_unicode"` |
| `nullify=value` | Tratar cadena específica como vacía | `prep:"nullify=NULL"` |
| `coerce=type` | Coerción de tipo (int, float, bool) | `prep:"coerce=int"` |
| `fix_scheme=scheme` | Agregar o corregir esquema de URL | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | Reemplazo basado en expresión regular | `prep:"regex_replace=\\d+:X"` |

## Etiquetas de Validación (`validate`)

Se pueden combinar múltiples etiquetas: `validate:"required,email"`

### Validadores Básicos

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `required` | El campo no debe estar vacío | `validate:"required"` |
| `omitempty` | Omitir validadores posteriores si el valor está vacío | `validate:"omitempty,email"` |
| `boolean` | Debe ser true, false, 0 o 1 | `validate:"boolean"` |

### Validadores de Tipo de Carácter

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `alpha` | Solo caracteres alfabéticos ASCII | `validate:"alpha"` |
| `alphaunicode` | Solo letras Unicode | `validate:"alphaunicode"` |
| `alphaspace` | Caracteres alfabéticos o espacios | `validate:"alphaspace"` |
| `alphanumeric` | Caracteres alfanuméricos ASCII | `validate:"alphanumeric"` |
| `alphanumunicode` | Letras o dígitos Unicode | `validate:"alphanumunicode"` |
| `numeric` | Entero válido | `validate:"numeric"` |
| `number` | Número válido (entero o decimal) | `validate:"number"` |
| `ascii` | Solo caracteres ASCII | `validate:"ascii"` |
| `printascii` | Caracteres ASCII imprimibles (0x20-0x7E) | `validate:"printascii"` |
| `multibyte` | Contiene caracteres multibyte | `validate:"multibyte"` |

### Validadores de Comparación Numérica

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `eq=N` | El valor es igual a N | `validate:"eq=100"` |
| `ne=N` | El valor no es igual a N | `validate:"ne=0"` |
| `gt=N` | El valor es mayor que N | `validate:"gt=0"` |
| `gte=N` | El valor es mayor o igual a N | `validate:"gte=1"` |
| `lt=N` | El valor es menor que N | `validate:"lt=100"` |
| `lte=N` | El valor es menor o igual a N | `validate:"lte=99"` |
| `min=N` | El valor es al menos N | `validate:"min=0"` |
| `max=N` | El valor es como máximo N | `validate:"max=100"` |
| `len=N` | Exactamente N caracteres | `validate:"len=10"` |

### Validadores de Cadenas

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `oneof=a b c` | El valor es uno de los permitidos | `validate:"oneof=active inactive"` |
| `lowercase` | El valor está todo en minúsculas | `validate:"lowercase"` |
| `uppercase` | El valor está todo en mayúsculas | `validate:"uppercase"` |
| `eq_ignore_case=value` | Igualdad sin distinción de mayúsculas/minúsculas | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | Desigualdad sin distinción de mayúsculas/minúsculas | `validate:"ne_ignore_case=no"` |

### Validadores de Contenido de Cadenas

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `startswith=prefix` | El valor empieza con prefijo | `validate:"startswith=http"` |
| `startsnotwith=prefix` | El valor no empieza con prefijo | `validate:"startsnotwith=_"` |
| `endswith=suffix` | El valor termina con sufijo | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | El valor no termina con sufijo | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | El valor contiene subcadena | `validate:"contains=@"` |
| `containsany=chars` | El valor contiene alguno de los caracteres | `validate:"containsany=abc"` |
| `containsrune=r` | El valor contiene el rune | `validate:"containsrune=@"` |
| `excludes=substr` | El valor no contiene subcadena | `validate:"excludes=admin"` |
| `excludesall=chars` | El valor no contiene ninguno de los caracteres | `validate:"excludesall=<>"` |
| `excludesrune=r` | El valor no contiene el rune | `validate:"excludesrune=$"` |

### Validadores de Formato

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `email` | Dirección de email válida | `validate:"email"` |
| `uri` | URI válido | `validate:"uri"` |
| `url` | URL válida | `validate:"url"` |
| `http_url` | URL HTTP o HTTPS válida | `validate:"http_url"` |
| `https_url` | URL HTTPS válida | `validate:"https_url"` |
| `url_encoded` | Cadena codificada URL | `validate:"url_encoded"` |
| `datauri` | Data URI válido | `validate:"datauri"` |
| `datetime=layout` | Datetime válido que coincide con layout de Go | `validate:"datetime=2006-01-02"` |
| `uuid` | UUID válido (cualquier versión) | `validate:"uuid"` |
| `uuid3` | UUID versión 3 válido | `validate:"uuid3"` |
| `uuid4` | UUID versión 4 válido | `validate:"uuid4"` |
| `uuid5` | UUID versión 5 válido | `validate:"uuid5"` |
| `ulid` | ULID válido | `validate:"ulid"` |
| `e164` | Número de teléfono E.164 válido | `validate:"e164"` |
| `latitude` | Latitud válida (-90 a 90) | `validate:"latitude"` |
| `longitude` | Longitud válida (-180 a 180) | `validate:"longitude"` |
| `hexadecimal` | Cadena hexadecimal válida | `validate:"hexadecimal"` |
| `hexcolor` | Código de color hexadecimal válido | `validate:"hexcolor"` |
| `rgb` | Color RGB válido | `validate:"rgb"` |
| `rgba` | Color RGBA válido | `validate:"rgba"` |
| `hsl` | Color HSL válido | `validate:"hsl"` |
| `hsla` | Color HSLA válido | `validate:"hsla"` |

### Validadores de Red

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `ip_addr` | Dirección IP válida (v4 o v6) | `validate:"ip_addr"` |
| `ip4_addr` | Dirección IPv4 válida | `validate:"ip4_addr"` |
| `ip6_addr` | Dirección IPv6 válida | `validate:"ip6_addr"` |
| `cidr` | Notación CIDR válida | `validate:"cidr"` |
| `cidrv4` | CIDR IPv4 válido | `validate:"cidrv4"` |
| `cidrv6` | CIDR IPv6 válido | `validate:"cidrv6"` |
| `mac` | Dirección MAC válida | `validate:"mac"` |
| `fqdn` | Nombre de dominio totalmente cualificado válido | `validate:"fqdn"` |
| `hostname` | Nombre de host válido (RFC 952) | `validate:"hostname"` |
| `hostname_rfc1123` | Nombre de host válido (RFC 1123) | `validate:"hostname_rfc1123"` |
| `hostname_port` | hostname:puerto válido | `validate:"hostname_port"` |

### Validadores de Campo Cruzado

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `eqfield=Field` | El valor es igual a otro campo | `validate:"eqfield=Password"` |
| `nefield=Field` | El valor no es igual a otro campo | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | El valor es mayor que otro campo | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | El valor es >= otro campo | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | El valor es menor que otro campo | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | El valor es <= otro campo | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | El valor contiene el valor de otro campo | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | El valor excluye el valor de otro campo | `validate:"fieldexcludes=Forbidden"` |

### Validadores de Requerimiento Condicional

| Etiqueta | Descripción | Ejemplo |
|----------|-------------|---------|
| `required_if=Field value` | Requerido si el campo es igual a value | `validate:"required_if=Status active"` |
| `required_unless=Field value` | Requerido a menos que el campo sea igual a value | `validate:"required_unless=Type guest"` |
| `required_with=Field` | Requerido si el campo está presente | `validate:"required_with=Email"` |
| `required_without=Field` | Requerido si el campo está ausente | `validate:"required_without=Phone"` |

**Ejemplos:**

```go
type User struct {
    Role    string
    // Profile es obligatorio cuando Role es "admin", opcional para otros roles
    Profile string `validate:"required_if=Role admin"`
    // Bio es obligatorio a menos que Role sea "guest"
    Bio     string `validate:"required_unless=Role guest"`
}

type Contact struct {
    Email string
    Phone string
    // Name es obligatorio cuando Email no está vacío
    Name  string `validate:"required_with=Email"`
    // Se debe proporcionar al menos Email o BackupEmail
    BackupEmail string `validate:"required_without=Email"`
}
```

## Formatos de Archivo Soportados

| Formato | Extensión | Extensiones Comprimidas |
|---------|-----------|------------------------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst`, `.csv.z`, `.csv.snappy`, `.csv.s2`, `.csv.lz4` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst`, `.tsv.z`, `.tsv.snappy`, `.tsv.s2`, `.tsv.lz4` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst`, `.ltsv.z`, `.ltsv.snappy`, `.ltsv.s2`, `.ltsv.lz4` |
| JSON | `.json` | `.json.gz`, `.json.bz2`, `.json.xz`, `.json.zst`, `.json.z`, `.json.snappy`, `.json.s2`, `.json.lz4` |
| JSONL | `.jsonl` | `.jsonl.gz`, `.jsonl.bz2`, `.jsonl.xz`, `.jsonl.zst`, `.jsonl.z`, `.jsonl.snappy`, `.jsonl.s2`, `.jsonl.lz4` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst`, `.xlsx.z`, `.xlsx.snappy`, `.xlsx.s2`, `.xlsx.lz4` |
| Parquet | `.parquet` | `.parquet.gz`, `.parquet.bz2`, `.parquet.xz`, `.parquet.zst`, `.parquet.z`, `.parquet.snappy`, `.parquet.s2`, `.parquet.lz4` |

### Formatos de Compresión Soportados

| Formato | Extensión | Biblioteca | Notas |
|---------|-----------|-----------|-------|
| gzip | `.gz` | compress/gzip | Biblioteca estándar |
| bzip2 | `.bz2` | compress/bzip2 | Biblioteca estándar |
| xz | `.xz` | github.com/ulikunitz/xz | Pure Go |
| zstd | `.zst` | github.com/klauspost/compress/zstd | Pure Go, alto rendimiento |
| zlib | `.z` | compress/zlib | Biblioteca estándar |
| snappy | `.snappy` | github.com/klauspost/compress/snappy | Pure Go, alto rendimiento |
| s2 | `.s2` | github.com/klauspost/compress/s2 | Compatible con Snappy, más rápido |
| lz4 | `.lz4` | github.com/pierrec/lz4/v4 | Pure Go |

**Nota sobre compresión Parquet**: La compresión externa (`.parquet.gz`, etc.) es para el archivo contenedor en sí. Los archivos Parquet también pueden usar compresión interna (Snappy, GZIP, LZ4, ZSTD) que es manejada de forma transparente por la biblioteca parquet-go.

## Integración con filesql

```go
// Procesar archivo con preprocesamiento y validación
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// Verificar errores de validación
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("Fila %d, Columna %s: %s", e.Row, e.Column, e.Message)
    }
}

// Pasar datos preprocesados a filesql usando el patrón Builder
ctx := context.Background()
builder := filesql.NewBuilder().
    AddReader(reader, "my_table", filesql.FileTypeCSV)

validatedBuilder, err := builder.Build(ctx)
if err != nil {
    return err
}

db, err := validatedBuilder.Open(ctx)
if err != nil {
    return err
}
defer db.Close()

// Ejecutar consultas SQL en los datos preprocesados
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## Opciones del Procesador

`NewProcessor` acepta opciones funcionales para personalizar el comportamiento:

### WithStrictTagParsing

Por defecto, los argumentos de tags inválidos (por ejemplo, `eq=abc` donde se espera un número) se ignoran silenciosamente. Habilite el modo estricto para detectar estas configuraciones incorrectas:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithStrictTagParsing())
var records []MyRecord

// Retorna un error si algún argumento de tag es inválido (ej: "eq=abc", "truncate=xyz")
_, _, err := processor.Process(input, &records)
```

### WithValidRowsOnly

Por defecto, la salida incluye todas las filas (válidas e inválidas). Use `WithValidRowsOnly` para filtrar la salida solo a filas válidas:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithValidRowsOnly())
var records []MyRecord

reader, result, err := processor.Process(input, &records)
// reader contiene solo las filas que pasaron todas las validaciones
// records contiene solo structs válidos
// result.RowCount incluye todas las filas; result.ValidRowCount tiene el conteo válido
// result.Errors aún reporta todos los fallos de validación
```

Las opciones se pueden combinar:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV,
    fileprep.WithStrictTagParsing(),
    fileprep.WithValidRowsOnly(),
)
```

## Consideraciones de Diseño

### Vinculación de Columnas por Nombre

Los campos struct se mapean a las columnas del archivo **por nombre**, no por posición. Los nombres de campo se convierten automáticamente a `snake_case` para coincidir con los encabezados de columna. El orden de las columnas en el archivo no importa.

```go
type User struct {
    UserName string `name:"user"`       // coincide con columna "user" (no "user_name")
    Email    string `name:"mail_addr"`  // coincide con columna "mail_addr" (no "email")
    Age      string                     // coincide con columna "age" (snake_case automático)
}
```

Si sus claves LTSV usan guiones (`user-id`) o las columnas Parquet/XLSX usan camelCase (`userId`), use la etiqueta `name` para especificar el nombre exacto de la columna.

Consulte [Antes de usar fileprep](#antes-de-usar-fileprep) para las reglas de sensibilidad a mayúsculas/minúsculas, el comportamiento de encabezados duplicados y el manejo de columnas faltantes.

### Uso de Memoria

fileprep carga el **archivo completo en memoria** para su procesamiento. Esto permite el acceso aleatorio y las operaciones de múltiples pasadas, pero tiene implicaciones para archivos grandes:

| Tamaño de Archivo | Memoria Aprox. | Recomendación |
|-------------------|----------------|---------------|
| < 100 MB | ~2-3x tamaño del archivo | Procesamiento directo |
| 100-500 MB | ~500 MB - 1.5 GB | Monitorear memoria, considerar fragmentación |
| > 500 MB | > 1.5 GB | Dividir archivos o usar alternativas de streaming |

Para entradas comprimidas (gzip, bzip2, xz, zstd, zlib, snappy, s2, lz4), el uso de memoria se basa en el tamaño **descomprimido**.

## Rendimiento

Resultados de benchmark procesando archivos CSV con un struct complejo que contiene 21 columnas. Cada campo usa múltiples etiquetas de preprocesamiento y validación:

**Etiquetas de preprocesamiento usadas:** trim, lowercase, uppercase, keep_digits, pad_left, strip_html, strip_newline, collapse_space, truncate, fix_scheme, default

**Etiquetas de validación usadas:** required, alpha, numeric, email, uuid, ip_addr, url, oneof, min, max, len, printascii, ascii, eqfield

| Registros | Tiempo | Memoria | Allocs/op |
|--------:|-----:|-------:|----------:|
| 100 | 0.6 ms | 0.9 MB | 7,654 |
| 1,000 | 6.1 ms | 9.6 MB | 74,829 |
| 10,000 | 69 ms | 101 MB | 746,266 |
| 50,000 | 344 ms | 498 MB | 3,690,281 |

```bash
# Benchmark rápido (100 y 1,000 registros)
make bench

# Benchmark completo (todos los tamaños incluyendo 50,000 registros)
make bench-all
```

*Probado en AMD Ryzen AI MAX+ 395, Go 1.24, Linux. Los resultados varían según el hardware.*

## Proyectos Relacionados o de Inspiración

- [nao1215/filesql](https://github.com/nao1215/filesql) - Driver SQL para CSV, TSV, LTSV, Parquet, Excel con soporte para gzip, bzip2, xz, zstd.
- [nao1215/fileframe](https://github.com/nao1215/fileframe) - API DataFrame para CSV/TSV/LTSV, Parquet, Excel.
- [nao1215/csv](https://github.com/nao1215/csv) - Lectura de CSV con validación y DataFrame simple en golang.
- [go-playground/validator](https://github.com/go-playground/validator) - Validación de Go Struct y Field, incluyendo Cross Field, Cross Struct, Map, Slice y Array diving.
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - go-header-csv es un codificador/decodificador de csv con encabezado.

## Contribuciones

¡Las contribuciones son bienvenidas! Por favor consulte la [Guía de Contribución](../../CONTRIBUTING.md) para más detalles.

## Soporte

Si encuentra útil este proyecto, por favor considere:

- Darle una estrella en GitHub - ayuda a otros a descubrir el proyecto
- [Convertirse en patrocinador](https://github.com/sponsors/nao1215) - su apoyo mantiene vivo el proyecto y motiva el desarrollo continuo

Su apoyo, ya sea a través de estrellas, patrocinios o contribuciones, es lo que impulsa este proyecto hacia adelante. ¡Gracias!

## Licencia

Este proyecto está licenciado bajo la Licencia MIT - vea el archivo [LICENSE](../../LICENSE) para más detalles.
