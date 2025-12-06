# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)

![fileprep-logo](./doc/images/fileprep-logo-small.png)

**fileprep** is a Go library for preprocessing and validating file data using struct tags.

## Why fileprep?

I developed [nao1215/filesql](https://github.com/nao1215/filesql), which allows you to execute SQL queries on files like CSV, TSV, LTSV, Parquet, and Excel. I also created [nao1215/csv](https://github.com/nao1215/csv) for CSV file validation.

While studying machine learning, I realized: "If I extend [nao1215/csv](https://github.com/nao1215/csv) to support the same file formats as [nao1215/filesql](https://github.com/nao1215/filesql), I could combine them to perform ETL-like operations." This idea led to the creation of **fileprep**—a library that bridges data preprocessing/validation with SQL-based file querying.

## Features

- Multiple file format support: CSV, TSV, LTSV, Parquet, Excel (.xlsx)
- Compression support: gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst)
- Struct tag-based preprocessing** (`prep` tag): trim, lowercase, uppercase, default values
- Struct tag-based validation (`validate` tag): required, and more
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
    "os"
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
| `pad_left=N,char` | Left-pad to N characters | `prep:"pad_left=5,0"` |
| `pad_right=N,char` | Right-pad to N characters | `prep:"pad_right=10, "` |

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
| `uuid` | Valid UUID | `validate:"uuid"` |

### Network Validators

| Tag | Description | Example |
|-----|-------------|---------|
| `ip_addr` | Valid IP address (v4 or v6) | `validate:"ip_addr"` |
| `ip4_addr` | Valid IPv4 address | `validate:"ip4_addr"` |
| `ip6_addr` | Valid IPv6 address | `validate:"ip6_addr"` |
| `cidr` | Valid CIDR notation | `validate:"cidr"` |
| `cidrv4` | Valid IPv4 CIDR | `validate:"cidrv4"` |
| `cidrv6` | Valid IPv6 CIDR | `validate:"cidrv6"` |
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

## Supported File Formats

| Format | Extension | Compressed |
|--------|-----------|------------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst` |
| Parquet | `.parquet` | - |

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

## Related Projects

- [nao1215/filesql](https://github.com/nao1215/filesql) - sql driver for CSV, TSV, LTSV, Parquet, Excel with gzip, bzip2, xz, zstd support.
- [nao1215/csv](https://github.com/nao1215/csv) - read csv with validation and simple DataFrame in golang.

## Contributing

Contributions are welcome! Please see the [Contributing Guide](./CONTRIBUTING.md) for more details.

## Support

If you find this project useful, please consider:

- Giving it a star on GitHub - it helps others discover the project
- [Becoming a sponsor](https://github.com/sponsors/nao1215) - your support keeps the project alive and motivates continued development

Your support, whether through stars, sponsorships, or contributions, is what drives this project forward. Thank you!

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
