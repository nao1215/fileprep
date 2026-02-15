# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[English](../../README.md) | [日本語](../ja/README.md) | [Español](../es/README.md) | [Français](../fr/README.md) | [한국어](../ko/README.md) | [中文](../zh-cn/README.md)

![fileprep-logo](../images/fileprep-logo-small.png)

**fileprep** — это библиотека Go для очистки, нормализации и валидации структурированных данных (CSV, TSV, LTSV, JSON, JSONL, Parquet и Excel) с использованием лёгких правил на основе struct-тегов. Поддерживает прозрачную работу с потоками gzip, bzip2, xz, zstd, zlib, snappy, s2 и lz4.

## Почему fileprep?

Я разработал [nao1215/filesql](https://github.com/nao1215/filesql), который позволяет выполнять SQL-запросы к файлам CSV, TSV, LTSV, Parquet и Excel. Также я создал [nao1215/csv](https://github.com/nao1215/csv) для валидации CSV-файлов.

Изучая машинное обучение, я понял: «Если расширить [nao1215/csv](https://github.com/nao1215/csv) для поддержки тех же форматов файлов, что и [nao1215/filesql](https://github.com/nao1215/filesql), можно объединить их для выполнения ETL-подобных операций». Эта идея привела к созданию **fileprep**: библиотеки, соединяющей предобработку/валидацию данных с SQL-запросами к файлам.

## Возможности

- Поддержка множества форматов: CSV, TSV, LTSV, JSON (.json), JSONL (.jsonl), Parquet, Excel (.xlsx)
- Поддержка сжатия: gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst), zlib (.z), snappy (.snappy), s2 (.s2), lz4 (.lz4)
- Привязка колонок по имени: поля автоматически соответствуют именам колонок в `snake_case`, настраивается через тег `name`
- Предобработка на основе struct-тегов (`prep`): trim, lowercase, uppercase, значения по умолчанию
- Валидация на основе struct-тегов (`validate`): required, omitempty и другие
- Опции процессора: `WithStrictTagParsing()` для обнаружения ошибок конфигурации тегов, `WithValidRowsOnly()` для фильтрации вывода
- Интеграция с [filesql](https://github.com/nao1215/filesql): возвращает `io.Reader` для прямого использования с filesql
- Детальные отчёты об ошибках: информация о строке и колонке для каждой ошибки

## Установка

```bash
go get github.com/nao1215/fileprep
```

## Требования

- Версия Go: 1.24 или выше
- Операционные системы:
  - Linux
  - macOS
  - Windows


## Быстрый старт

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// User представляет запись пользователя с предобработкой и валидацией
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
        fmt.Printf("Ошибка: %v\n", err)
        return
    }

    fmt.Printf("Обработано %d строк, %d валидных\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("Имя: %q, Email: %q\n", user.Name, user.Email)
    }

    // reader можно передать напрямую в filesql
    _ = reader
}
```

Вывод:
```
Обработано 2 строк, 2 валидных
Имя: "John Doe", Email: "john@example.com"
Имя: "Jane Smith", Email: "jane@example.com"
```

## Перед использованием fileprep

### JSON/JSONL использует единственную колонку "data"

Файлы JSON/JSONL парсятся в единственную колонку `"data"`. Каждый элемент массива (JSON) или строка (JSONL) становится одной строкой, содержащей необработанную JSON-строку.

```go
type JSONRecord struct {
    Data string `name:"data" prep:"trim" validate:"required"`
}
```

Вывод всегда в формате компактного JSONL. Если тег prep нарушает структуру JSON, `Process` возвращает `ErrInvalidJSONAfterPrep`. Если все строки становятся пустыми, возвращается `ErrEmptyJSONOutput`.

### Сопоставление колонок чувствительно к регистру

`UserName` сопоставляется с `user_name` через авто snake_case. Заголовки типа `User_Name`, `USER_NAME`, `userName` **не** совпадут. Используйте тег `name`, когда заголовки отличаются:

```go
type Record struct {
    UserName string                 // совпадает только с "user_name"
    Email    string `name:"EMAIL"`  // совпадает точно с "EMAIL"
}
```

### Дублирующиеся заголовки: первая колонка выигрывает

Если файл содержит `id,id,name`, первая колонка `id` используется для привязки. Вторая игнорируется.

### Отсутствующие колонки становятся пустыми строками

Если колонка не существует для поля структуры, значение будет `""`. Добавьте `validate:"required"` для обнаружения этого при парсинге.

### Excel: обрабатывается только первый лист

Файлы `.xlsx` с несколькими листами будут молча игнорировать все листы после первого.

## Расширенные примеры

### Комплексная предобработка и валидация данных

Этот пример демонстрирует всю мощь fileprep: комбинирование нескольких препроцессоров и валидаторов для очистки и валидации реальных «грязных» данных.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee представляет данные сотрудника с комплексной предобработкой и валидацией
type Employee struct {
    // ID: дополнить до 6 цифр, должен быть числовым
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // Имя: очистить пробелы, обязательное буквенное с пробелами
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // Email: нормализовать в нижний регистр, проверить формат
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // Отдел: нормализовать регистр, должен быть одним из разрешённых значений
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // Зарплата: оставить только цифры, проверить диапазон
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // Телефон: извлечь цифры, проверить формат E.164 после добавления кода страны
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // Дата начала: проверить формат даты
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // ID руководителя: обязателен, если отдел не HR
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // Сайт: исправить отсутствующую схему, проверить URL
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // «Грязные» реальные CSV-данные
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
        fmt.Printf("Критическая ошибка: %v\n", err)
        return
    }

    fmt.Printf("=== Результат обработки ===\n")
    fmt.Printf("Всего строк: %d, Валидных: %d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("Сотрудник %d:\n", i+1)
        fmt.Printf("  ID:          %s\n", emp.EmployeeID)
        fmt.Printf("  Имя:         %s\n", emp.FullName)
        fmt.Printf("  Email:       %s\n", emp.Email)
        fmt.Printf("  Отдел:       %s\n", emp.Department)
        fmt.Printf("  Зарплата:    %s\n", emp.Salary)
        fmt.Printf("  Телефон:     %s\n", emp.Phone)
        fmt.Printf("  Дата начала: %s\n", emp.StartDate)
        fmt.Printf("  ID руков.:   %s\n", emp.ManagerID)
        fmt.Printf("  Сайт:        %s\n\n", emp.Website)
    }
}
```

Вывод:
```
=== Результат обработки ===
Всего строк: 4, Валидных: 4

Сотрудник 1:
  ID:          000042
  Имя:         John Doe
  Email:       john.doe@company.com
  Отдел:       ENGINEERING
  Зарплата:    75000
  Телефон:     +15551234567
  Дата начала: 2023-01-15
  ID руков.:   000001
  Сайт:        https://company.com/john

Сотрудник 2:
  ID:          000007
  Имя:         Jane Smith
  Email:       jane@company.com
  Отдел:       SALES
  Зарплата:    120000
  Телефон:     +15559876543
  Дата начала: 2022-06-01
  ID руков.:   000002
  Сайт:        https://www.linkedin.com/in/jane

Сотрудник 3:
  ID:          000123
  Имя:         Bob Wilson
  Email:       bob.wilson@company.com
  Отдел:       HR
  Зарплата:    45000
  Телефон:     +15551112222
  Дата начала: 2024-03-20
  ID руков.:   000000
  Сайт:

Сотрудник 4:
  ID:          000099
  Имя:         Alice Brown
  Email:       alice@company.com
  Отдел:       MARKETING
  Зарплата:    88500
  Телефон:     +15554443333
  Дата начала: 2023-09-10
  ID руков.:   000003
  Сайт:        https://alice.dev
```


### Детальные отчёты об ошибках

При ошибках валидации fileprep предоставляет точную информацию об ошибках, включая номер строки, имя колонки и конкретную причину ошибки.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order представляет заказ со строгими правилами валидации
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
    // CSV с множественными ошибками валидации
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
        fmt.Printf("Критическая ошибка: %v\n", err)
        return
    }

    fmt.Printf("=== Отчёт о валидации ===\n")
    fmt.Printf("Всего строк:      %d\n", result.RowCount)
    fmt.Printf("Валидных:         %d\n", result.ValidRowCount)
    fmt.Printf("Невалидных:       %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("Всего ошибок:     %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== Детали ошибок ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("Строка %d, Колонка '%s': %s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

Вывод:
```
=== Отчёт о валидации ===
Всего строк:      4
Валидных:         1
Невалидных:       3
Всего ошибок:     23

=== Детали ошибок ===
Строка 2, Колонка 'order_id': value must be a valid UUID version 4
Строка 2, Колонка 'customer_id': value must be numeric
Строка 2, Колонка 'email': value must be a valid email address
Строка 2, Колонка 'amount': value must be greater than 0
Строка 2, Колонка 'currency': value must have exactly 3 characters
Строка 2, Колонка 'country': value must have exactly 2 characters
Строка 2, Колонка 'order_date': value must be a valid datetime in format: 2006-01-02
Строка 2, Колонка 'ip_address': value must be a valid IP address
Строка 2, Колонка 'promo_code': value must contain only alphanumeric characters
Строка 2, Колонка 'quantity': value must be greater than or equal to 1
Строка 2, Колонка 'unit_price': value must be greater than 0
Строка 2, Колонка 'ship_date': value must be greater than field OrderDate
Строка 2, Колонка 'total_check': value must equal field Amount
Строка 3, Колонка 'customer_id': value is required
Строка 3, Колонка 'email': value must be a valid email address
Строка 3, Колонка 'amount': value must be less than or equal to 10000
Строка 3, Колонка 'currency': value must have exactly 3 characters
Строка 3, Колонка 'country': value must contain only alphabetic characters
Строка 3, Колонка 'order_date': value must be a valid datetime in format: 2006-01-02
Строка 3, Колонка 'quantity': value must be less than or equal to 100
Строка 3, Колонка 'unit_price': value must be greater than 0
Строка 3, Колонка 'ship_date': value must be greater than field OrderDate
Строка 4, Колонка 'ship_date': value must be greater than field OrderDate
```

## Теги предобработки (`prep`)

Можно комбинировать несколько тегов: `prep:"trim,lowercase,default=N/A"`

### Базовые препроцессоры

| Тег | Описание | Пример |
|-----|----------|--------|
| `trim` | Удалить пробелы в начале/конце | `prep:"trim"` |
| `ltrim` | Удалить пробелы в начале | `prep:"ltrim"` |
| `rtrim` | Удалить пробелы в конце | `prep:"rtrim"` |
| `lowercase` | Преобразовать в нижний регистр | `prep:"lowercase"` |
| `uppercase` | Преобразовать в верхний регистр | `prep:"uppercase"` |
| `default=value` | Установить значение по умолчанию, если пусто | `prep:"default=N/A"` |

### Преобразование строк

| Тег | Описание | Пример |
|-----|----------|--------|
| `replace=old:new` | Заменить все вхождения | `prep:"replace=;:,"` |
| `prefix=value` | Добавить строку в начало | `prep:"prefix=ID_"` |
| `suffix=value` | Добавить строку в конец | `prep:"suffix=_END"` |
| `truncate=N` | Ограничить N символами | `prep:"truncate=100"` |
| `strip_html` | Удалить HTML-теги | `prep:"strip_html"` |
| `strip_newline` | Удалить переносы строк (LF, CRLF, CR) | `prep:"strip_newline"` |
| `collapse_space` | Сжать множественные пробелы в один | `prep:"collapse_space"` |

### Фильтрация символов

| Тег | Описание | Пример |
|-----|----------|--------|
| `remove_digits` | Удалить все цифры | `prep:"remove_digits"` |
| `remove_alpha` | Удалить все буквы | `prep:"remove_alpha"` |
| `keep_digits` | Оставить только цифры | `prep:"keep_digits"` |
| `keep_alpha` | Оставить только буквы | `prep:"keep_alpha"` |
| `trim_set=chars` | Удалить указанные символы с обоих концов | `prep:"trim_set=@#$"` |

### Выравнивание

| Тег | Описание | Пример |
|-----|----------|--------|
| `pad_left=N:char` | Дополнить слева до N символов | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | Дополнить справа до N символов | `prep:"pad_right=10: "` |

### Расширенная предобработка

| Тег | Описание | Пример |
|-----|----------|--------|
| `normalize_unicode` | Нормализовать Unicode в формат NFC | `prep:"normalize_unicode"` |
| `nullify=value` | Считать определённую строку пустой | `prep:"nullify=NULL"` |
| `coerce=type` | Преобразование типа (int, float, bool) | `prep:"coerce=int"` |
| `fix_scheme=scheme` | Добавить/исправить схему URL | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | Замена по регулярному выражению | `prep:"regex_replace=\\d+:X"` |

## Теги валидации (`validate`)

Можно комбинировать несколько тегов: `validate:"required,email"`

### Базовые валидаторы

| Тег | Описание | Пример |
|-----|----------|--------|
| `required` | Поле не должно быть пустым | `validate:"required"` |
| `omitempty` | Пропустить последующие валидаторы, если значение пустое | `validate:"omitempty,email"` |
| `boolean` | Должно быть true, false, 0 или 1 | `validate:"boolean"` |

### Валидаторы типа символов

| Тег | Описание | Пример |
|-----|----------|--------|
| `alpha` | Только ASCII-буквы | `validate:"alpha"` |
| `alphaunicode` | Только Unicode-буквы | `validate:"alphaunicode"` |
| `alphaspace` | Буквы или пробелы | `validate:"alphaspace"` |
| `alphanumeric` | ASCII-буквы и цифры | `validate:"alphanumeric"` |
| `alphanumunicode` | Unicode-буквы или цифры | `validate:"alphanumunicode"` |
| `numeric` | Целое число | `validate:"numeric"` |
| `number` | Число (целое или десятичное) | `validate:"number"` |
| `ascii` | Только ASCII-символы | `validate:"ascii"` |
| `printascii` | Печатные ASCII-символы (0x20-0x7E) | `validate:"printascii"` |
| `multibyte` | Содержит многобайтовые символы | `validate:"multibyte"` |

### Числовые валидаторы сравнения

| Тег | Описание | Пример |
|-----|----------|--------|
| `eq=N` | Значение равно N | `validate:"eq=100"` |
| `ne=N` | Значение не равно N | `validate:"ne=0"` |
| `gt=N` | Значение больше N | `validate:"gt=0"` |
| `gte=N` | Значение больше или равно N | `validate:"gte=1"` |
| `lt=N` | Значение меньше N | `validate:"lt=100"` |
| `lte=N` | Значение меньше или равно N | `validate:"lte=99"` |
| `min=N` | Значение не меньше N | `validate:"min=0"` |
| `max=N` | Значение не больше N | `validate:"max=100"` |
| `len=N` | Ровно N символов | `validate:"len=10"` |

### Строковые валидаторы

| Тег | Описание | Пример |
|-----|----------|--------|
| `oneof=a b c` | Значение одно из разрешённых | `validate:"oneof=active inactive"` |
| `lowercase` | Значение в нижнем регистре | `validate:"lowercase"` |
| `uppercase` | Значение в верхнем регистре | `validate:"uppercase"` |
| `eq_ignore_case=value` | Равенство без учёта регистра | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | Неравенство без учёта регистра | `validate:"ne_ignore_case=no"` |

### Валидаторы содержимого строк

| Тег | Описание | Пример |
|-----|----------|--------|
| `startswith=prefix` | Значение начинается с prefix | `validate:"startswith=http"` |
| `startsnotwith=prefix` | Значение не начинается с prefix | `validate:"startsnotwith=_"` |
| `endswith=suffix` | Значение заканчивается на suffix | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | Значение не заканчивается на suffix | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | Значение содержит подстроку | `validate:"contains=@"` |
| `containsany=chars` | Значение содержит любой из символов | `validate:"containsany=abc"` |
| `containsrune=r` | Значение содержит символ | `validate:"containsrune=@"` |
| `excludes=substr` | Значение не содержит подстроку | `validate:"excludes=admin"` |
| `excludesall=chars` | Значение не содержит ни одного из символов | `validate:"excludesall=<>"` |
| `excludesrune=r` | Значение не содержит символ | `validate:"excludesrune=$"` |

### Валидаторы формата

| Тег | Описание | Пример |
|-----|----------|--------|
| `email` | Валидный email-адрес | `validate:"email"` |
| `uri` | Валидный URI | `validate:"uri"` |
| `url` | Валидный URL | `validate:"url"` |
| `http_url` | Валидный HTTP или HTTPS URL | `validate:"http_url"` |
| `https_url` | Валидный HTTPS URL | `validate:"https_url"` |
| `url_encoded` | URL-кодированная строка | `validate:"url_encoded"` |
| `datauri` | Валидный data URI | `validate:"datauri"` |
| `datetime=layout` | Валидная дата по формату Go | `validate:"datetime=2006-01-02"` |
| `uuid` | Валидный UUID (любая версия) | `validate:"uuid"` |
| `uuid3` | Валидный UUID версии 3 | `validate:"uuid3"` |
| `uuid4` | Валидный UUID версии 4 | `validate:"uuid4"` |
| `uuid5` | Валидный UUID версии 5 | `validate:"uuid5"` |
| `ulid` | Валидный ULID | `validate:"ulid"` |
| `e164` | Валидный номер E.164 | `validate:"e164"` |
| `latitude` | Валидная широта (-90 до 90) | `validate:"latitude"` |
| `longitude` | Валидная долгота (-180 до 180) | `validate:"longitude"` |
| `hexadecimal` | Валидная шестнадцатеричная строка | `validate:"hexadecimal"` |
| `hexcolor` | Валидный hex-код цвета | `validate:"hexcolor"` |
| `rgb` | Валидный RGB-цвет | `validate:"rgb"` |
| `rgba` | Валидный RGBA-цвет | `validate:"rgba"` |
| `hsl` | Валидный HSL-цвет | `validate:"hsl"` |
| `hsla` | Валидный HSLA-цвет | `validate:"hsla"` |

### Сетевые валидаторы

| Тег | Описание | Пример |
|-----|----------|--------|
| `ip_addr` | Валидный IP-адрес (v4 или v6) | `validate:"ip_addr"` |
| `ip4_addr` | Валидный IPv4-адрес | `validate:"ip4_addr"` |
| `ip6_addr` | Валидный IPv6-адрес | `validate:"ip6_addr"` |
| `cidr` | Валидная CIDR-нотация | `validate:"cidr"` |
| `cidrv4` | Валидный IPv4 CIDR | `validate:"cidrv4"` |
| `cidrv6` | Валидный IPv6 CIDR | `validate:"cidrv6"` |
| `mac` | Валидный MAC-адрес | `validate:"mac"` |
| `fqdn` | Валидное полное доменное имя | `validate:"fqdn"` |
| `hostname` | Валидное имя хоста (RFC 952) | `validate:"hostname"` |
| `hostname_rfc1123` | Валидное имя хоста (RFC 1123) | `validate:"hostname_rfc1123"` |
| `hostname_port` | Валидный hostname:port | `validate:"hostname_port"` |

### Межполевые валидаторы

| Тег | Описание | Пример |
|-----|----------|--------|
| `eqfield=Field` | Значение равно другому полю | `validate:"eqfield=Password"` |
| `nefield=Field` | Значение не равно другому полю | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | Значение больше другого поля | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | Значение >= другого поля | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | Значение меньше другого поля | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | Значение <= другого поля | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | Значение содержит значение другого поля | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | Значение исключает значение другого поля | `validate:"fieldexcludes=Forbidden"` |

### Условные валидаторы обязательности

| Тег | Описание | Пример |
|-----|----------|--------|
| `required_if=Field value` | Обязательно, если поле равно значению | `validate:"required_if=Status active"` |
| `required_unless=Field value` | Обязательно, если поле не равно значению | `validate:"required_unless=Type guest"` |
| `required_with=Field` | Обязательно, если поле присутствует | `validate:"required_with=Email"` |
| `required_without=Field` | Обязательно, если поле отсутствует | `validate:"required_without=Phone"` |

**Примеры:**

```go
type User struct {
    Role    string
    // Profile обязателен, когда Role равен "admin", необязателен для других ролей
    Profile string `validate:"required_if=Role admin"`
    // Bio обязателен, если Role не равен "guest"
    Bio     string `validate:"required_unless=Role guest"`
}

type Contact struct {
    Email string
    Phone string
    // Name обязателен, когда Email не пуст
    Name  string `validate:"required_with=Email"`
    // Хотя бы один из Email или BackupEmail должен быть указан
    BackupEmail string `validate:"required_without=Email"`
}
```

## Поддерживаемые форматы файлов

| Формат | Расширение | Сжатые форматы |
|--------|------------|----------------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst`, `.csv.z`, `.csv.snappy`, `.csv.s2`, `.csv.lz4` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst`, `.tsv.z`, `.tsv.snappy`, `.tsv.s2`, `.tsv.lz4` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst`, `.ltsv.z`, `.ltsv.snappy`, `.ltsv.s2`, `.ltsv.lz4` |
| JSON | `.json` | `.json.gz`, `.json.bz2`, `.json.xz`, `.json.zst`, `.json.z`, `.json.snappy`, `.json.s2`, `.json.lz4` |
| JSONL | `.jsonl` | `.jsonl.gz`, `.jsonl.bz2`, `.jsonl.xz`, `.jsonl.zst`, `.jsonl.z`, `.jsonl.snappy`, `.jsonl.s2`, `.jsonl.lz4` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst`, `.xlsx.z`, `.xlsx.snappy`, `.xlsx.s2`, `.xlsx.lz4` |
| Parquet | `.parquet` | `.parquet.gz`, `.parquet.bz2`, `.parquet.xz`, `.parquet.zst`, `.parquet.z`, `.parquet.snappy`, `.parquet.s2`, `.parquet.lz4` |

### Поддерживаемые форматы сжатия

| Формат | Расширение | Описание |
|--------|------------|----------|
| gzip | `.gz` | GNU zip сжатие, широко используется |
| bzip2 | `.bz2` | Блочное сжатие с отличной степенью |
| xz | `.xz` | Высокопроизводительное LZMA2 сжатие |
| zstd | `.zst` | Zstandard от Facebook |
| zlib | `.z` | Стандартное DEFLATE сжатие |
| snappy | `.snappy` | Быстрое сжатие от Google |
| s2 | `.s2` | Улучшенное расширение Snappy |
| lz4 | `.lz4` | Сверхбыстрое сжатие |

**Примечание о сжатии Parquet**: Внешнее сжатие (`.parquet.gz` и т.д.) применяется к самому файлу-контейнеру. Файлы Parquet также могут использовать внутреннее сжатие (Snappy, GZIP, LZ4, ZSTD), которое прозрачно обрабатывается библиотекой parquet-go.

## Интеграция с filesql

```go
// Обработка файла с предобработкой и валидацией
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// Проверка ошибок валидации
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("Строка %d, Колонка %s: %s", e.Row, e.Column, e.Message)
    }
}

// Передача предобработанных данных в filesql с использованием Builder
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

// Выполнение SQL-запросов к предобработанным данным
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## Опции процессора

`NewProcessor` принимает функциональные опции для настройки поведения:

### WithStrictTagParsing

По умолчанию недопустимые аргументы тегов (например, `eq=abc`, где ожидается число) молча игнорируются. Включите строгий режим для обнаружения таких ошибок конфигурации:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithStrictTagParsing())
var records []MyRecord

// Возвращает ошибку, если аргумент тега недопустим (например, "eq=abc", "truncate=xyz")
_, _, err := processor.Process(input, &records)
```

### WithValidRowsOnly

По умолчанию вывод включает все строки (как валидные, так и невалидные). Используйте `WithValidRowsOnly` для фильтрации вывода только валидными строками:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithValidRowsOnly())
var records []MyRecord

reader, result, err := processor.Process(input, &records)
// reader содержит только строки, прошедшие все валидации
// records содержит только валидные структуры
// result.RowCount включает все строки; result.ValidRowCount содержит количество валидных
// result.Errors по-прежнему сообщает обо всех ошибках валидации
```

Опции можно комбинировать:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV,
    fileprep.WithStrictTagParsing(),
    fileprep.WithValidRowsOnly(),
)
```

## Особенности проектирования

### Привязка колонок по имени

Поля структуры сопоставляются с колонками файла **по имени**, а не по позиции. Имена полей автоматически преобразуются в `snake_case` для соответствия заголовкам колонок. Порядок колонок не имеет значения.

```go
type User struct {
    UserName string `name:"user"`       // соответствует колонке "user" (не "user_name")
    Email    string `name:"mail_addr"`  // соответствует колонке "mail_addr" (не "email")
    Age      string                     // соответствует колонке "age" (авто snake_case)
}
```

Если ваши ключи LTSV используют дефисы (`user-id`) или колонки Parquet/XLSX используют camelCase (`userId`), используйте тег `name` для указания точного имени колонки.

Смотрите [Перед использованием fileprep](#перед-использованием-fileprep) для правил чувствительности к регистру, поведения дублирующихся заголовков и обработки отсутствующих колонок.

### Использование памяти

fileprep загружает **весь файл в память** для обработки. Это позволяет произвольный доступ и многопроходные операции, но имеет последствия для больших файлов:

| Размер файла | Примерная память | Рекомендация |
|--------------|------------------|--------------|
| < 100 МБ | ~2-3x размера файла | Прямая обработка |
| 100-500 МБ | ~500 МБ - 1.5 ГБ | Мониторинг памяти, рассмотреть разбиение |
| > 500 МБ | > 1.5 ГБ | Разделить файлы или использовать потоковые альтернативы |

Для сжатых входных данных (gzip, bzip2, xz, zstd, zlib, snappy, s2, lz4) использование памяти основано на **распакованном** размере.

## Производительность

Результаты бенчмарков обработки CSV-файлов со сложной структурой, содержащей 21 колонку. Каждое поле использует несколько тегов предобработки и валидации:

**Используемые теги предобработки:** trim, lowercase, uppercase, keep_digits, pad_left, strip_html, strip_newline, collapse_space, truncate, fix_scheme, default

**Используемые теги валидации:** required, alpha, numeric, email, uuid, ip_addr, url, oneof, min, max, len, printascii, ascii, eqfield

| Записей | Время | Память | Аллокаций |
|--------:|------:|-------:|----------:|
| 100 | 0.6 мс | 0.9 МБ | 7,654 |
| 1,000 | 6.1 мс | 9.6 МБ | 74,829 |
| 10,000 | 69 мс | 101 МБ | 746,266 |
| 50,000 | 344 мс | 498 МБ | 3,690,281 |

```bash
# Быстрый бенчмарк (100 и 1,000 записей)
make bench

# Полный бенчмарк (все размеры включая 50,000 записей)
make bench-all
```

*Тестировано на AMD Ryzen AI MAX+ 395, Go 1.24, Linux. Результаты зависят от оборудования.*

## Связанные и вдохновившие проекты

- [nao1215/filesql](https://github.com/nao1215/filesql) - SQL-драйвер для CSV, TSV, LTSV, Parquet, Excel с поддержкой gzip, bzip2, xz, zstd.
- [nao1215/fileframe](https://github.com/nao1215/fileframe) - DataFrame API для CSV/TSV/LTSV, Parquet, Excel.
- [nao1215/csv](https://github.com/nao1215/csv) - Чтение CSV с валидацией и простой DataFrame на Go.
- [go-playground/validator](https://github.com/go-playground/validator) - Валидация структур и полей Go, включая кросс-полевую, кросс-структурную, Map, Slice и Array валидацию
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - go-header-csv — кодировщик/декодировщик CSV с заголовком.

## Участие в разработке

Мы приветствуем вклад в проект! Подробнее см. [Contributing Guide](../../CONTRIBUTING.md).

## Поддержка

Если вы находите этот проект полезным, пожалуйста, рассмотрите:

- Поставить звезду на GitHub — это помогает другим найти проект
- [Стать спонсором](https://github.com/sponsors/nao1215) — ваша поддержка помогает поддерживать проект и мотивирует на дальнейшую разработку

Ваша поддержка, будь то звёзды, спонсорство или вклад в код, движет этот проект вперёд. Спасибо!

## Лицензия

Этот проект лицензирован под лицензией MIT — подробности см. в файле [LICENSE](../../LICENSE).
