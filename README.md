# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[日本語](doc/ja/README.md) | [Español](doc/es/README.md) | [Français](doc/fr/README.md) | [한국어](doc/ko/README.md) | [Русский](doc/ru/README.md) | [中文](doc/zh-cn/README.md)

![fileprep-logo](./doc/images/fileprep-logo-small.png)

**fileprep** is a Go library for cleaning, normalizing, and validating structured data—CSV, TSV, LTSV, JSON, JSONL, Parquet, and Excel—through lightweight struct-tag rules, with seamless support for gzip, bzip2, xz, zstd, zlib, snappy, s2, and lz4 streams.

## Why fileprep?

I developed [nao1215/filesql](https://github.com/nao1215/filesql), which allows you to execute SQL queries on files like CSV, TSV, LTSV, Parquet, and Excel. I also created [nao1215/csv](https://github.com/nao1215/csv) for CSV file validation.

While studying machine learning, I realized: "If I extend [nao1215/csv](https://github.com/nao1215/csv) to support the same file formats as [nao1215/filesql](https://github.com/nao1215/filesql), I could combine them to perform ETL-like operations." This idea led to the creation of **fileprep**—a library that bridges data preprocessing/validation with SQL-based file querying.

## Features

- Multiple file format support: CSV, TSV, LTSV, JSON (.json), JSONL (.jsonl), Parquet, Excel (.xlsx)
- Compression support: gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst), zlib (.z), snappy (.snappy), s2 (.s2), lz4 (.lz4)
- Name-based column binding: Fields auto-match `snake_case` column names, customizable via `name` tag
- Struct tag-based preprocessing (`prep` tag): trim, lowercase, uppercase, default values
- Struct tag-based validation (`validate` tag): required, omitempty, and more
- Processor options: `WithStrictTagParsing()` for catching tag misconfigurations, `WithValidRowsOnly()` for filtering output
- Seamless [filesql](https://github.com/nao1215/filesql) integration: Returns `io.Reader` for direct use with filesql
- Detailed error reporting: Row and column information for each error

## Installation

```bash
go get github.com/nao1215/fileprep
```

## Requirements

- Go Version: 1.24 or later
- Operating Systems:
  - Linux
  - macOS  
  - Windows


## Quick Start

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// User represents a user record with preprocessing and validation
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

    fmt.Printf("Processed %d rows, %d valid\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("Name: %q, Email: %q\n", user.Name, user.Email)
    }

    // reader can be passed directly to filesql
    _ = reader
}
```

Output:
```
Processed 2 rows, 2 valid
Name: "John Doe", Email: "john@example.com"
Name: "Jane Smith", Email: "jane@example.com"
```

## Before Using fileprep

### JSON/JSONL uses a single "data" column

JSON/JSONL files are parsed into a single column named `"data"`. Each array element (JSON) or line (JSONL) becomes one row containing the raw JSON string.

```go
type JSONRecord struct {
    Data string `name:"data" prep:"trim" validate:"required"`
}
```

Output is always compact JSONL. If a prep tag breaks the JSON structure, `Process` returns `ErrInvalidJSONAfterPrep`. If all rows end up empty, it returns `ErrEmptyJSONOutput`.

### Column matching is case-sensitive

`UserName` maps to `user_name` via auto snake_case. Headers like `User_Name`, `USER_NAME`, `userName` do **not** match. Use the `name` tag when headers differ:

```go
type Record struct {
    UserName string                 // matches "user_name" only
    Email    string `name:"EMAIL"`  // matches "EMAIL" exactly
}
```

### Duplicate headers: first column wins

If a file has `id,id,name`, the first `id` column is used for binding. The second is ignored.

### Missing columns become empty strings

If a column doesn't exist for a struct field, the value is `""`. Add `validate:"required"` to catch this at parse time.

### Excel: only the first sheet is processed

Multi-sheet `.xlsx` files will silently ignore all sheets after the first.

## Advanced Examples

### Complex Data Preprocessing and Validation

This example demonstrates the full power of fileprep: combining multiple preprocessors and validators to clean and validate real-world messy data.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee represents employee data with comprehensive preprocessing and validation
type Employee struct {
    // ID: pad to 6 digits, must be numeric
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // Name: clean whitespace, required alphabetic with spaces
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // Email: normalize to lowercase, validate format
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // Department: normalize case, must be one of allowed values
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // Salary: keep only digits, validate range
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // Phone: extract digits, validate E.164 format after adding country code
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // Start date: validate datetime format
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // Manager ID: required only if department is not HR
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // Website: fix missing scheme, validate URL
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // Messy real-world CSV data
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
        fmt.Printf("Fatal error: %v\n", err)
        return
    }

    fmt.Printf("=== Processing Result ===\n")
    fmt.Printf("Total rows: %d, Valid rows: %d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("Employee %d:\n", i+1)
        fmt.Printf("  ID:         %s\n", emp.EmployeeID)
        fmt.Printf("  Name:       %s\n", emp.FullName)
        fmt.Printf("  Email:      %s\n", emp.Email)
        fmt.Printf("  Department: %s\n", emp.Department)
        fmt.Printf("  Salary:     %s\n", emp.Salary)
        fmt.Printf("  Phone:      %s\n", emp.Phone)
        fmt.Printf("  Start Date: %s\n", emp.StartDate)
        fmt.Printf("  Manager ID: %s\n", emp.ManagerID)
        fmt.Printf("  Website:    %s\n\n", emp.Website)
    }
}
```

Output:
```
=== Processing Result ===
Total rows: 4, Valid rows: 4

Employee 1:
  ID:         000042
  Name:       John Doe
  Email:      john.doe@company.com
  Department: ENGINEERING
  Salary:     75000
  Phone:      +15551234567
  Start Date: 2023-01-15
  Manager ID: 000001
  Website:    https://company.com/john

Employee 2:
  ID:         000007
  Name:       Jane Smith
  Email:      jane@company.com
  Department: SALES
  Salary:     120000
  Phone:      +15559876543
  Start Date: 2022-06-01
  Manager ID: 000002
  Website:    https://www.linkedin.com/in/jane

Employee 3:
  ID:         000123
  Name:       Bob Wilson
  Email:      bob.wilson@company.com
  Department: HR
  Salary:     45000
  Phone:      +15551112222
  Start Date: 2024-03-20
  Manager ID: 000000
  Website:

Employee 4:
  ID:         000099
  Name:       Alice Brown
  Email:      alice@company.com
  Department: MARKETING
  Salary:     88500
  Phone:      +15554443333
  Start Date: 2023-09-10
  Manager ID: 000003
  Website:    https://alice.dev
```


### Detailed Error Reporting

When validation fails, fileprep provides precise error information including row number, column name, and specific validation failure reason.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order represents an order with strict validation rules
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
    // CSV with multiple validation errors
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
        fmt.Printf("Fatal error: %v\n", err)
        return
    }

    fmt.Printf("=== Validation Report ===\n")
    fmt.Printf("Total rows:     %d\n", result.RowCount)
    fmt.Printf("Valid rows:     %d\n", result.ValidRowCount)
    fmt.Printf("Invalid rows:   %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("Total errors:   %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== Error Details ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("Row %d, Column '%s': %s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

Output:
```
=== Validation Report ===
Total rows:     4
Valid rows:     1
Invalid rows:   3
Total errors:   23

=== Error Details ===
Row 2, Column 'order_id': value must be a valid UUID version 4
Row 2, Column 'customer_id': value must be numeric
Row 2, Column 'email': value must be a valid email address
Row 2, Column 'amount': value must be greater than 0
Row 2, Column 'currency': value must have exactly 3 characters
Row 2, Column 'country': value must have exactly 2 characters
Row 2, Column 'order_date': value must be a valid datetime in format: 2006-01-02
Row 2, Column 'ip_address': value must be a valid IP address
Row 2, Column 'promo_code': value must contain only alphanumeric characters
Row 2, Column 'quantity': value must be greater than or equal to 1
Row 2, Column 'unit_price': value must be greater than 0
Row 2, Column 'ship_date': value must be greater than field OrderDate
Row 2, Column 'total_check': value must equal field Amount
Row 3, Column 'customer_id': value is required
Row 3, Column 'email': value must be a valid email address
Row 3, Column 'amount': value must be less than or equal to 10000
Row 3, Column 'currency': value must have exactly 3 characters
Row 3, Column 'country': value must contain only alphabetic characters
Row 3, Column 'order_date': value must be a valid datetime in format: 2006-01-02
Row 3, Column 'quantity': value must be less than or equal to 100
Row 3, Column 'unit_price': value must be greater than 0
Row 3, Column 'ship_date': value must be greater than field OrderDate
Row 4, Column 'ship_date': value must be greater than field OrderDate
```

## Preprocessing Tags (`prep`)

Multiple tags can be combined: `prep:"trim,lowercase,default=N/A"`

### Basic Preprocessors

| Tag | Description | Example |
|-----|-------------|---------|
| `trim` | Remove leading/trailing whitespace | `prep:"trim"` |
| `ltrim` | Remove leading whitespace | `prep:"ltrim"` |
| `rtrim` | Remove trailing whitespace | `prep:"rtrim"` |
| `lowercase` | Convert to lowercase | `prep:"lowercase"` |
| `uppercase` | Convert to uppercase | `prep:"uppercase"` |
| `default=value` | Set default if empty | `prep:"default=N/A"` |

### String Transformation

| Tag | Description | Example |
|-----|-------------|---------|
| `replace=old:new` | Replace all occurrences | `prep:"replace=;:,"` |
| `prefix=value` | Prepend string to value | `prep:"prefix=ID_"` |
| `suffix=value` | Append string to value | `prep:"suffix=_END"` |
| `truncate=N` | Limit to N characters | `prep:"truncate=100"` |
| `strip_html` | Remove HTML tags | `prep:"strip_html"` |
| `strip_newline` | Remove newlines (LF, CRLF, CR) | `prep:"strip_newline"` |
| `collapse_space` | Collapse multiple spaces into one | `prep:"collapse_space"` |

### Character Filtering

| Tag | Description | Example |
|-----|-------------|---------|
| `remove_digits` | Remove all digits | `prep:"remove_digits"` |
| `remove_alpha` | Remove all alphabetic characters | `prep:"remove_alpha"` |
| `keep_digits` | Keep only digits | `prep:"keep_digits"` |
| `keep_alpha` | Keep only alphabetic characters | `prep:"keep_alpha"` |
| `trim_set=chars` | Remove specified characters from both ends | `prep:"trim_set=@#$"` |

### Padding

| Tag | Description | Example |
|-----|-------------|---------|
| `pad_left=N:char` | Left-pad to N characters | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | Right-pad to N characters | `prep:"pad_right=10: "` |

### Advanced Preprocessors

| Tag | Description | Example |
|-----|-------------|---------|
| `normalize_unicode` | Normalize Unicode to NFC form | `prep:"normalize_unicode"` |
| `nullify=value` | Treat specific string as empty | `prep:"nullify=NULL"` |
| `coerce=type` | Type coercion (int, float, bool) | `prep:"coerce=int"` |
| `fix_scheme=scheme` | Add or fix URL scheme | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | Regex-based replacement | `prep:"regex_replace=\\d+:X"` |

## Validation Tags (`validate`)

Multiple tags can be combined: `validate:"required,email"`

### Basic Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must not be empty | `validate:"required"` |
| `omitempty` | Skip subsequent validators if value is empty | `validate:"omitempty,email"` |
| `boolean` | Must be true, false, 0, or 1 | `validate:"boolean"` |

### Character Type Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `alpha` | ASCII alphabetic characters only | `validate:"alpha"` |
| `alphaunicode` | Unicode letters only | `validate:"alphaunicode"` |
| `alphaspace` | Alphabetic characters or spaces | `validate:"alphaspace"` |
| `alphanumeric` | ASCII alphanumeric characters | `validate:"alphanumeric"` |
| `alphanumunicode` | Unicode letters or digits | `validate:"alphanumunicode"` |
| `numeric` | Valid integer | `validate:"numeric"` |
| `number` | Valid number (integer or decimal) | `validate:"number"` |
| `ascii` | ASCII characters only | `validate:"ascii"` |
| `printascii` | Printable ASCII characters (0x20-0x7E) | `validate:"printascii"` |
| `multibyte` | Contains multibyte characters | `validate:"multibyte"` |

### Numeric Comparison Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `eq=N` | Value equals N | `validate:"eq=100"` |
| `ne=N` | Value not equals N | `validate:"ne=0"` |
| `gt=N` | Value greater than N | `validate:"gt=0"` |
| `gte=N` | Value greater than or equal to N | `validate:"gte=1"` |
| `lt=N` | Value less than N | `validate:"lt=100"` |
| `lte=N` | Value less than or equal to N | `validate:"lte=99"` |
| `min=N` | Value at least N | `validate:"min=0"` |
| `max=N` | Value at most N | `validate:"max=100"` |
| `len=N` | Exactly N characters | `validate:"len=10"` |

### String Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `oneof=a b c` | Value is one of the allowed values | `validate:"oneof=active inactive"` |
| `lowercase` | Value is all lowercase | `validate:"lowercase"` |
| `uppercase` | Value is all uppercase | `validate:"uppercase"` |
| `eq_ignore_case=value` | Case-insensitive equality | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | Case-insensitive not equal | `validate:"ne_ignore_case=no"` |

### String Content Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `startswith=prefix` | Value starts with prefix | `validate:"startswith=http"` |
| `startsnotwith=prefix` | Value does not start with prefix | `validate:"startsnotwith=_"` |
| `endswith=suffix` | Value ends with suffix | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | Value does not end with suffix | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | Value contains substring | `validate:"contains=@"` |
| `containsany=chars` | Value contains any of the chars | `validate:"containsany=abc"` |
| `containsrune=r` | Value contains the rune | `validate:"containsrune=@"` |
| `excludes=substr` | Value does not contain substring | `validate:"excludes=admin"` |
| `excludesall=chars` | Value does not contain any of the chars | `validate:"excludesall=<>"` |
| `excludesrune=r` | Value does not contain the rune | `validate:"excludesrune=$"` |

### Format Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `email` | Valid email address | `validate:"email"` |
| `uri` | Valid URI | `validate:"uri"` |
| `url` | Valid URL | `validate:"url"` |
| `http_url` | Valid HTTP or HTTPS URL | `validate:"http_url"` |
| `https_url` | Valid HTTPS URL | `validate:"https_url"` |
| `url_encoded` | URL encoded string | `validate:"url_encoded"` |
| `datauri` | Valid data URI | `validate:"datauri"` |
| `datetime=layout` | Valid datetime matching Go layout | `validate:"datetime=2006-01-02"` |
| `uuid` | Valid UUID (any version) | `validate:"uuid"` |
| `uuid3` | Valid UUID version 3 | `validate:"uuid3"` |
| `uuid4` | Valid UUID version 4 | `validate:"uuid4"` |
| `uuid5` | Valid UUID version 5 | `validate:"uuid5"` |
| `ulid` | Valid ULID | `validate:"ulid"` |
| `e164` | Valid E.164 phone number | `validate:"e164"` |
| `latitude` | Valid latitude (-90 to 90) | `validate:"latitude"` |
| `longitude` | Valid longitude (-180 to 180) | `validate:"longitude"` |
| `hexadecimal` | Valid hexadecimal string | `validate:"hexadecimal"` |
| `hexcolor` | Valid hex color code | `validate:"hexcolor"` |
| `rgb` | Valid RGB color | `validate:"rgb"` |
| `rgba` | Valid RGBA color | `validate:"rgba"` |
| `hsl` | Valid HSL color | `validate:"hsl"` |
| `hsla` | Valid HSLA color | `validate:"hsla"` |

### Network Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `ip_addr` | Valid IP address (v4 or v6) | `validate:"ip_addr"` |
| `ip4_addr` | Valid IPv4 address | `validate:"ip4_addr"` |
| `ip6_addr` | Valid IPv6 address | `validate:"ip6_addr"` |
| `cidr` | Valid CIDR notation | `validate:"cidr"` |
| `cidrv4` | Valid IPv4 CIDR | `validate:"cidrv4"` |
| `cidrv6` | Valid IPv6 CIDR | `validate:"cidrv6"` |
| `mac` | Valid MAC address | `validate:"mac"` |
| `fqdn` | Valid fully qualified domain name | `validate:"fqdn"` |
| `hostname` | Valid hostname (RFC 952) | `validate:"hostname"` |
| `hostname_rfc1123` | Valid hostname (RFC 1123) | `validate:"hostname_rfc1123"` |
| `hostname_port` | Valid hostname:port | `validate:"hostname_port"` |

### Cross-Field Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `eqfield=Field` | Value equals another field | `validate:"eqfield=Password"` |
| `nefield=Field` | Value not equals another field | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | Value greater than another field | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | Value >= another field | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | Value less than another field | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | Value <= another field | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | Value contains another field's value | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | Value excludes another field's value | `validate:"fieldexcludes=Forbidden"` |

### Conditional Required Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `required_if=Field value` | Required if field equals value | `validate:"required_if=Status active"` |
| `required_unless=Field value` | Required unless field equals value | `validate:"required_unless=Type guest"` |
| `required_with=Field` | Required if field is present | `validate:"required_with=Email"` |
| `required_without=Field` | Required if field is absent | `validate:"required_without=Phone"` |

**Examples:**

```go
type User struct {
    Role    string
    // Profile is required when Role is "admin", optional for other roles
    Profile string `validate:"required_if=Role admin"`
    // Bio is required unless Role is "guest"
    Bio     string `validate:"required_unless=Role guest"`
}

type Contact struct {
    Email string
    Phone string
    // Name is required when Email is non-empty
    Name  string `validate:"required_with=Email"`
    // At least one of Email or BackupEmail must be provided
    BackupEmail string `validate:"required_without=Email"`
}
```

## Supported File Formats

| Format | Extension | Compressed Extensions |
|--------|-----------|----------------------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst`, `.csv.z`, `.csv.snappy`, `.csv.s2`, `.csv.lz4` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst`, `.tsv.z`, `.tsv.snappy`, `.tsv.s2`, `.tsv.lz4` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst`, `.ltsv.z`, `.ltsv.snappy`, `.ltsv.s2`, `.ltsv.lz4` |
| JSON | `.json` | `.json.gz`, `.json.bz2`, `.json.xz`, `.json.zst`, `.json.z`, `.json.snappy`, `.json.s2`, `.json.lz4` |
| JSONL | `.jsonl` | `.jsonl.gz`, `.jsonl.bz2`, `.jsonl.xz`, `.jsonl.zst`, `.jsonl.z`, `.jsonl.snappy`, `.jsonl.s2`, `.jsonl.lz4` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst`, `.xlsx.z`, `.xlsx.snappy`, `.xlsx.s2`, `.xlsx.lz4` |
| Parquet | `.parquet` | `.parquet.gz`, `.parquet.bz2`, `.parquet.xz`, `.parquet.zst`, `.parquet.z`, `.parquet.snappy`, `.parquet.s2`, `.parquet.lz4` |

### Supported Compression Formats

| Format | Extension | Library | Notes |
|--------|-----------|---------|-------|
| gzip | `.gz` | compress/gzip | Standard library |
| bzip2 | `.bz2` | compress/bzip2 | Standard library |
| xz | `.xz` | github.com/ulikunitz/xz | Pure Go |
| zstd | `.zst` | github.com/klauspost/compress/zstd | Pure Go, high performance |
| zlib | `.z` | compress/zlib | Standard library |
| snappy | `.snappy` | github.com/klauspost/compress/snappy | Pure Go, high performance |
| s2 | `.s2` | github.com/klauspost/compress/s2 | Snappy-compatible, faster |
| lz4 | `.lz4` | github.com/pierrec/lz4/v4 | Pure Go |

**Note on Parquet compression**: The external compression (`.parquet.gz`, etc.) is for the container file itself. Parquet files may also use internal compression (Snappy, GZIP, LZ4, ZSTD) which is handled transparently by the parquet-go library.

## Integration with filesql

```go
// Process file with preprocessing and validation
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// Check for validation errors
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("Row %d, Column %s: %s", e.Row, e.Column, e.Message)
    }
}

// Pass preprocessed data to filesql using Builder pattern
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

// Execute SQL queries on preprocessed data
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## Processor Options

`NewProcessor` accepts functional options to customize behavior:

### WithStrictTagParsing

By default, invalid tag arguments (e.g., `eq=abc` where a number is expected) are silently ignored. Enable strict mode to catch these misconfigurations:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithStrictTagParsing())
var records []MyRecord

// Returns an error if any tag argument is invalid (e.g., "eq=abc", "truncate=xyz")
_, _, err := processor.Process(input, &records)
```

### WithValidRowsOnly

By default, the output includes all rows (valid and invalid). Use `WithValidRowsOnly` to filter the output to only valid rows:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithValidRowsOnly())
var records []MyRecord

reader, result, err := processor.Process(input, &records)
// reader contains only rows that passed all validations
// records contains only valid structs
// result.RowCount includes all rows; result.ValidRowCount has the valid count
// result.Errors still reports all validation failures
```

Options can be combined:

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV,
    fileprep.WithStrictTagParsing(),
    fileprep.WithValidRowsOnly(),
)
```

## Design Considerations

### Name-Based Column Binding

Struct fields are mapped to file columns **by name**, not by position. Field names are automatically converted to `snake_case` to match column headers. Column order in the file does not matter.

```go
type User struct {
    UserName string `name:"user"`       // matches "user" column (not "user_name")
    Email    string `name:"mail_addr"`  // matches "mail_addr" column (not "email")
    Age      string                     // matches "age" column (auto snake_case)
}
```

If your LTSV keys use hyphens (`user-id`) or Parquet/XLSX columns use camelCase (`userId`), use the `name` tag to specify the exact column name.

See [Before Using fileprep](#before-using-fileprep) for case-sensitivity rules, duplicate header behavior, and missing column handling.

### Memory Usage

fileprep loads the **entire file into memory** for processing. This enables random access and multi-pass operations but has implications for large files:

| File Size | Approx. Memory | Recommendation |
|-----------|----------------|----------------|
| < 100 MB | ~2-3x file size | Direct processing |
| 100-500 MB | ~500 MB - 1.5 GB | Monitor memory, consider chunking |
| > 500 MB | > 1.5 GB | Split files or use streaming alternatives |

For compressed inputs (gzip, bzip2, xz, zstd, zlib, snappy, s2, lz4), memory usage is based on **decompressed** size.

## Performance

Benchmark results processing CSV files with a complex struct containing 21 columns. Each field uses multiple preprocessing and validation tags:

**Preprocessing tags used:** trim, lowercase, uppercase, keep_digits, pad_left, strip_html, strip_newline, collapse_space, truncate, fix_scheme, default

**Validation tags used:** required, alpha, numeric, email, uuid, ip_addr, url, oneof, min, max, len, printascii, ascii, eqfield

| Records | Time | Memory | Allocs/op |
|--------:|-----:|-------:|----------:|
| 100 | 0.6 ms | 0.9 MB | 7,654 |
| 1,000 | 6.1 ms | 9.6 MB | 74,829 |
| 10,000 | 69 ms | 101 MB | 746,266 |
| 50,000 | 344 ms | 498 MB | 3,690,281 |

```bash
# Quick benchmark (100 and 1,000 records)
make bench

# Full benchmark (all sizes including 50,000 records)
make bench-all
```

*Tested on AMD Ryzen AI MAX+ 395, Go 1.24, Linux. Results vary by hardware.*

## Related or inspired Projects

- [nao1215/filesql](https://github.com/nao1215/filesql) - sql driver for CSV, TSV, LTSV, Parquet, Excel with gzip, bzip2, xz, zstd support.
- [nao1215/fileframe](https://github.com/nao1215/fileframe) - DataFrame API for CSV/TSV/LTSV, Parquet, Excel. 
- [nao1215/csv](https://github.com/nao1215/csv) - read csv with validation and simple DataFrame in golang.
- [go-playground/validator](https://github.com/go-playground/validator) - Go Struct and Field validation, including Cross Field, Cross Struct, Map, Slice and Array diving
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - go-header-csv is encoder/decoder csv with a header.

## Contributing

Contributions are welcome! Please see the [Contributing Guide](./CONTRIBUTING.md) for more details.

## Support

If you find this project useful, please consider:

- Giving it a star on GitHub - it helps others discover the project
- [Becoming a sponsor](https://github.com/sponsors/nao1215) - your support keeps the project alive and motivates continued development

Your support, whether through stars, sponsorships, or contributions, is what drives this project forward. Thank you!

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
