# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[English](../../README.md) | [日本語](../ja/README.md) | [Español](../es/README.md) | [Français](../fr/README.md) | [Русский](../ru/README.md) | [中文](../zh-cn/README.md)

![fileprep-logo](../images/fileprep-logo-small.png)

**fileprep**은 CSV, TSV, LTSV, Parquet, Excel 등의 구조화된 데이터를 경량 struct 태그 규칙으로 정리, 정규화, 검증하기 위한 Go 라이브러리입니다. gzip, bzip2, xz, zstd 스트림을 원활하게 지원합니다.

## 왜 fileprep인가?

저는 CSV, TSV, LTSV, Parquet, Excel 파일에 SQL 쿼리를 실행할 수 있는 [nao1215/filesql](https://github.com/nao1215/filesql)을 개발했습니다. 또한 CSV 파일 검증을 위해 [nao1215/csv](https://github.com/nao1215/csv)도 만들었습니다.

머신러닝을 공부하면서, "[nao1215/csv](https://github.com/nao1215/csv)를 [nao1215/filesql](https://github.com/nao1215/filesql)과 동일한 파일 형식으로 확장하면, 둘을 결합하여 ETL과 같은 작업을 수행할 수 있다"는 것을 깨달았습니다. 이 아이디어가 **fileprep**의 탄생으로 이어졌습니다: 데이터 전처리/검증과 SQL 기반 파일 쿼리를 연결하는 라이브러리입니다.

## 기능

- 다중 파일 형식 지원: CSV, TSV, LTSV, Parquet, Excel (.xlsx)
- 압축 지원: gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst)
- 이름 기반 컬럼 바인딩: 필드는 자동으로 `snake_case` 컬럼명에 매칭, `name` 태그로 커스터마이즈 가능
- struct 태그 기반 전처리 (`prep` 태그): trim, lowercase, uppercase, 기본값 등
- struct 태그 기반 검증 (`validate` 태그): required 등
- [filesql](https://github.com/nao1215/filesql)과의 원활한 통합: filesql에서 직접 사용할 수 있는 `io.Reader` 반환
- 상세한 에러 보고: 각 에러의 행과 열 정보

## 설치

```bash
go get github.com/nao1215/fileprep
```

## 요구 사항

- Go 버전: 1.24 이상
- 지원 OS:
  - Linux
  - macOS
  - Windows


## 빠른 시작

```go
package main

import (
    "fmt"
    "os"
    "strings"

    "github.com/nao1215/fileprep"
)

// User는 전처리 및 검증이 포함된 사용자 레코드를 나타냅니다
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

    fmt.Printf("처리 완료: %d행 중 %d행이 유효\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("Name: %q, Email: %q\n", user.Name, user.Email)
    }

    // reader는 filesql에 직접 전달할 수 있습니다
    _ = reader
}
```

출력:
```
처리 완료: 2행 중 2행이 유효
Name: "John Doe", Email: "john@example.com"
Name: "Jane Smith", Email: "jane@example.com"
```

## 고급 예제

### 복잡한 데이터 전처리 및 검증

이 예제는 fileprep의 전체 기능을 보여줍니다: 여러 전처리기와 검증기를 결합하여 실제 세계의 지저분한 데이터를 정리하고 검증합니다.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee는 포괄적인 전처리 및 검증이 포함된 직원 데이터를 나타냅니다
type Employee struct {
    // ID: 6자리로 패딩, 숫자여야 함
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // 이름: 공백 정리, 필수 알파벳 및 공백
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // 이메일: 소문자로 정규화, 형식 검증
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // 부서: 대문자로 정규화, 허용된 값 중 하나여야 함
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // 급여: 숫자만 추출, 범위 검증
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // 전화번호: 숫자 추출, 국가 코드 추가 후 E.164 형식 검증
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // 시작일: datetime 형식 검증
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // 관리자 ID: 부서가 HR이 아닌 경우에만 필수
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // 웹사이트: 누락된 스킴 수정, URL 검증
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // 지저분한 실제 세계의 CSV 데이터
    csvData := `id,name,email,department,salary,phone,start_date,manager_id,website
  42,  John   Doe  ,JOHN.DOE@COMPANY.COM,engineering,$75,000,555-123-4567,2023-01-15,000001,company.com/john
7,Jane Smith,jane@COMPANY.com,  Sales  ,"$120,000",(555) 987-6543,2022-06-01,000002,WWW.LINKEDIN.COM/in/jane
123,Bob Wilson,bob.wilson@company.com,HR,45000,555.111.2222,2024-03-20,,
99,Alice Brown,alice@company.com,Marketing,$88500,555-444-3333,2023-09-10,000003,https://alice.dev
`

    processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
    var employees []Employee

    _, result, err := processor.Process(strings.NewReader(csvData), &employees)
    if err != nil {
        fmt.Printf("치명적 오류: %v\n", err)
        return
    }

    fmt.Printf("=== 처리 결과 ===\n")
    fmt.Printf("총 행 수: %d, 유효 행 수: %d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("직원 %d:\n", i+1)
        fmt.Printf("  ID:         %s\n", emp.EmployeeID)
        fmt.Printf("  이름:       %s\n", emp.FullName)
        fmt.Printf("  이메일:     %s\n", emp.Email)
        fmt.Printf("  부서:       %s\n", emp.Department)
        fmt.Printf("  급여:       %s\n", emp.Salary)
        fmt.Printf("  전화번호:   %s\n", emp.Phone)
        fmt.Printf("  시작일:     %s\n", emp.StartDate)
        fmt.Printf("  관리자 ID:  %s\n", emp.ManagerID)
        fmt.Printf("  웹사이트:   %s\n\n", emp.Website)
    }
}
```

출력:
```
=== 처리 결과 ===
총 행 수: 4, 유효 행 수: 4

직원 1:
  ID:         000042
  이름:       John Doe
  이메일:     john.doe@company.com
  부서:       ENGINEERING
  급여:       75000
  전화번호:   +15551234567
  시작일:     2023-01-15
  관리자 ID:  000001
  웹사이트:   https://company.com/john

직원 2:
  ID:         000007
  이름:       Jane Smith
  이메일:     jane@company.com
  부서:       SALES
  급여:       120000
  전화번호:   +15559876543
  시작일:     2022-06-01
  관리자 ID:  000002
  웹사이트:   https://www.linkedin.com/in/jane

직원 3:
  ID:         000123
  이름:       Bob Wilson
  이메일:     bob.wilson@company.com
  부서:       HR
  급여:       45000
  전화번호:   +15551112222
  시작일:     2024-03-20
  관리자 ID:  000000
  웹사이트:

직원 4:
  ID:         000099
  이름:       Alice Brown
  이메일:     alice@company.com
  부서:       MARKETING
  급여:       88500
  전화번호:   +15554443333
  시작일:     2023-09-10
  관리자 ID:  000003
  웹사이트:   https://alice.dev
```


### 상세한 에러 보고

검증이 실패하면 fileprep은 행 번호, 컬럼 이름 및 구체적인 검증 실패 이유를 포함한 정확한 에러 정보를 제공합니다.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order는 엄격한 검증 규칙이 있는 주문을 나타냅니다
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
    // 여러 검증 에러가 있는 CSV
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
        fmt.Printf("치명적 오류: %v\n", err)
        return
    }

    fmt.Printf("=== 검증 보고서 ===\n")
    fmt.Printf("총 행 수:     %d\n", result.RowCount)
    fmt.Printf("유효 행 수:   %d\n", result.ValidRowCount)
    fmt.Printf("무효 행 수:   %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("총 에러 수:   %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== 에러 세부 정보 ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("행 %d, 컬럼 '%s': %s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

출력:
```
=== 검증 보고서 ===
총 행 수:     4
유효 행 수:   1
무효 행 수:   3
총 에러 수:   23

=== 에러 세부 정보 ===
행 2, 컬럼 'order_id': value must be a valid UUID version 4
행 2, 컬럼 'customer_id': value must be numeric
행 2, 컬럼 'email': value must be a valid email address
행 2, 컬럼 'amount': value must be greater than 0
행 2, 컬럼 'currency': value must have exactly 3 characters
행 2, 컬럼 'country': value must have exactly 2 characters
행 2, 컬럼 'order_date': value must be a valid datetime in format: 2006-01-02
행 2, 컬럼 'ip_address': value must be a valid IP address
행 2, 컬럼 'promo_code': value must contain only alphanumeric characters
행 2, 컬럼 'quantity': value must be greater than or equal to 1
행 2, 컬럼 'unit_price': value must be greater than 0
행 2, 컬럼 'ship_date': value must be greater than field OrderDate
행 2, 컬럼 'total_check': value must equal field Amount
행 3, 컬럼 'customer_id': value is required
행 3, 컬럼 'email': value must be a valid email address
행 3, 컬럼 'amount': value must be less than or equal to 10000
행 3, 컬럼 'currency': value must have exactly 3 characters
행 3, 컬럼 'country': value must contain only alphabetic characters
행 3, 컬럼 'order_date': value must be a valid datetime in format: 2006-01-02
행 3, 컬럼 'quantity': value must be less than or equal to 100
행 3, 컬럼 'unit_price': value must be greater than 0
행 3, 컬럼 'ship_date': value must be greater than field OrderDate
행 4, 컬럼 'ship_date': value must be greater than field OrderDate
```

## 전처리 태그 (`prep`)

여러 태그를 조합할 수 있습니다: `prep:"trim,lowercase,default=N/A"`

### 기본 전처리

| 태그 | 설명 | 예시 |
|------|------|------|
| `trim` | 앞뒤 공백 제거 | `prep:"trim"` |
| `ltrim` | 앞쪽 공백 제거 | `prep:"ltrim"` |
| `rtrim` | 뒤쪽 공백 제거 | `prep:"rtrim"` |
| `lowercase` | 소문자로 변환 | `prep:"lowercase"` |
| `uppercase` | 대문자로 변환 | `prep:"uppercase"` |
| `default=value` | 비어있을 경우 기본값 설정 | `prep:"default=N/A"` |

### 문자열 변환

| 태그 | 설명 | 예시 |
|------|------|------|
| `replace=old:new` | 모든 항목 치환 | `prep:"replace=;:,"` |
| `prefix=value` | 문자열 앞에 추가 | `prep:"prefix=ID_"` |
| `suffix=value` | 문자열 뒤에 추가 | `prep:"suffix=_END"` |
| `truncate=N` | N자로 제한 | `prep:"truncate=100"` |
| `strip_html` | HTML 태그 제거 | `prep:"strip_html"` |
| `strip_newline` | 줄바꿈 제거 (LF, CRLF, CR) | `prep:"strip_newline"` |
| `collapse_space` | 연속 공백을 하나로 축소 | `prep:"collapse_space"` |

### 문자 필터링

| 태그 | 설명 | 예시 |
|------|------|------|
| `remove_digits` | 모든 숫자 제거 | `prep:"remove_digits"` |
| `remove_alpha` | 모든 알파벳 제거 | `prep:"remove_alpha"` |
| `keep_digits` | 숫자만 유지 | `prep:"keep_digits"` |
| `keep_alpha` | 알파벳만 유지 | `prep:"keep_alpha"` |
| `trim_set=chars` | 지정 문자를 양끝에서 제거 | `prep:"trim_set=@#$"` |

### 패딩

| 태그 | 설명 | 예시 |
|------|------|------|
| `pad_left=N:char` | N자까지 왼쪽 패딩 | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | N자까지 오른쪽 패딩 | `prep:"pad_right=10: "` |

### 고급 전처리

| 태그 | 설명 | 예시 |
|------|------|------|
| `normalize_unicode` | Unicode를 NFC 형식으로 정규화 | `prep:"normalize_unicode"` |
| `nullify=value` | 특정 문자열을 빈 값으로 처리 | `prep:"nullify=NULL"` |
| `coerce=type` | 타입 변환 (int, float, bool) | `prep:"coerce=int"` |
| `fix_scheme=scheme` | URL 스킴 추가/수정 | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | 정규식으로 치환 | `prep:"regex_replace=\\d+:X"` |

## 검증 태그 (`validate`)

여러 태그를 조합할 수 있습니다: `validate:"required,email"`

### 기본 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `required` | 필드가 비어있으면 안 됨 | `validate:"required"` |
| `boolean` | true, false, 0, 또는 1이어야 함 | `validate:"boolean"` |

### 문자 타입 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `alpha` | ASCII 알파벳 문자만 | `validate:"alpha"` |
| `alphaunicode` | Unicode 문자만 | `validate:"alphaunicode"` |
| `alphaspace` | 알파벳 문자 또는 공백 | `validate:"alphaspace"` |
| `alphanumeric` | ASCII 영숫자 | `validate:"alphanumeric"` |
| `alphanumunicode` | Unicode 문자 또는 숫자 | `validate:"alphanumunicode"` |
| `numeric` | 유효한 정수 | `validate:"numeric"` |
| `number` | 유효한 숫자 (정수 또는 소수) | `validate:"number"` |
| `ascii` | ASCII 문자만 | `validate:"ascii"` |
| `printascii` | 인쇄 가능한 ASCII 문자 (0x20-0x7E) | `validate:"printascii"` |
| `multibyte` | 멀티바이트 문자 포함 | `validate:"multibyte"` |

### 숫자 비교 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `eq=N` | 값이 N과 같음 | `validate:"eq=100"` |
| `ne=N` | 값이 N과 같지 않음 | `validate:"ne=0"` |
| `gt=N` | 값이 N보다 큼 | `validate:"gt=0"` |
| `gte=N` | 값이 N 이상 | `validate:"gte=1"` |
| `lt=N` | 값이 N 미만 | `validate:"lt=100"` |
| `lte=N` | 값이 N 이하 | `validate:"lte=99"` |
| `min=N` | 값이 최소 N | `validate:"min=0"` |
| `max=N` | 값이 최대 N | `validate:"max=100"` |
| `len=N` | 정확히 N자 | `validate:"len=10"` |

### 문자열 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `oneof=a b c` | 허용된 값 중 하나 | `validate:"oneof=active inactive"` |
| `lowercase` | 모두 소문자 | `validate:"lowercase"` |
| `uppercase` | 모두 대문자 | `validate:"uppercase"` |
| `eq_ignore_case=value` | 대소문자 무시 동등 | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | 대소문자 무시 부등 | `validate:"ne_ignore_case=no"` |

### 문자열 내용 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `startswith=prefix` | 접두사로 시작 | `validate:"startswith=http"` |
| `startsnotwith=prefix` | 접두사로 시작하지 않음 | `validate:"startsnotwith=_"` |
| `endswith=suffix` | 접미사로 끝남 | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | 접미사로 끝나지 않음 | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | 부분 문자열 포함 | `validate:"contains=@"` |
| `containsany=chars` | 지정 문자 중 하나 포함 | `validate:"containsany=abc"` |
| `containsrune=r` | 지정 룬 포함 | `validate:"containsrune=@"` |
| `excludes=substr` | 부분 문자열 미포함 | `validate:"excludes=admin"` |
| `excludesall=chars` | 지정 문자 모두 미포함 | `validate:"excludesall=<>"` |
| `excludesrune=r` | 지정 룬 미포함 | `validate:"excludesrune=$"` |

### 형식 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `email` | 유효한 이메일 주소 | `validate:"email"` |
| `uri` | 유효한 URI | `validate:"uri"` |
| `url` | 유효한 URL | `validate:"url"` |
| `http_url` | 유효한 HTTP 또는 HTTPS URL | `validate:"http_url"` |
| `https_url` | 유효한 HTTPS URL | `validate:"https_url"` |
| `url_encoded` | URL 인코딩된 문자열 | `validate:"url_encoded"` |
| `datauri` | 유효한 데이터 URI | `validate:"datauri"` |
| `datetime=layout` | Go 레이아웃에 맞는 유효한 날짜시간 | `validate:"datetime=2006-01-02"` |
| `uuid` | 유효한 UUID (모든 버전) | `validate:"uuid"` |
| `uuid3` | 유효한 UUID 버전 3 | `validate:"uuid3"` |
| `uuid4` | 유효한 UUID 버전 4 | `validate:"uuid4"` |
| `uuid5` | 유효한 UUID 버전 5 | `validate:"uuid5"` |
| `ulid` | 유효한 ULID | `validate:"ulid"` |
| `e164` | 유효한 E.164 전화번호 | `validate:"e164"` |
| `latitude` | 유효한 위도 (-90 ~ 90) | `validate:"latitude"` |
| `longitude` | 유효한 경도 (-180 ~ 180) | `validate:"longitude"` |
| `hexadecimal` | 유효한 16진수 문자열 | `validate:"hexadecimal"` |
| `hexcolor` | 유효한 16진수 색상 코드 | `validate:"hexcolor"` |
| `rgb` | 유효한 RGB 색상 | `validate:"rgb"` |
| `rgba` | 유효한 RGBA 색상 | `validate:"rgba"` |
| `hsl` | 유효한 HSL 색상 | `validate:"hsl"` |
| `hsla` | 유효한 HSLA 색상 | `validate:"hsla"` |

### 네트워크 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `ip_addr` | 유효한 IP 주소 (v4 또는 v6) | `validate:"ip_addr"` |
| `ip4_addr` | 유효한 IPv4 주소 | `validate:"ip4_addr"` |
| `ip6_addr` | 유효한 IPv6 주소 | `validate:"ip6_addr"` |
| `cidr` | 유효한 CIDR 표기법 | `validate:"cidr"` |
| `cidrv4` | 유효한 IPv4 CIDR | `validate:"cidrv4"` |
| `cidrv6` | 유효한 IPv6 CIDR | `validate:"cidrv6"` |
| `mac` | 유효한 MAC 주소 | `validate:"mac"` |
| `fqdn` | 유효한 완전한 도메인 이름 | `validate:"fqdn"` |
| `hostname` | 유효한 호스트명 (RFC 952) | `validate:"hostname"` |
| `hostname_rfc1123` | 유효한 호스트명 (RFC 1123) | `validate:"hostname_rfc1123"` |
| `hostname_port` | 유효한 호스트명:포트 | `validate:"hostname_port"` |

### 필드 간 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `eqfield=Field` | 다른 필드와 값이 같음 | `validate:"eqfield=Password"` |
| `nefield=Field` | 다른 필드와 값이 다름 | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | 다른 필드보다 큼 | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | 다른 필드 이상 | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | 다른 필드보다 작음 | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | 다른 필드 이하 | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | 다른 필드의 값 포함 | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | 다른 필드의 값 미포함 | `validate:"fieldexcludes=Forbidden"` |

### 조건부 필수 검증자

| 태그 | 설명 | 예시 |
|------|------|------|
| `required_if=Field value` | 필드가 value와 같으면 필수 | `validate:"required_if=Status active"` |
| `required_unless=Field value` | 필드가 value와 같지 않으면 필수 | `validate:"required_unless=Type guest"` |
| `required_with=Field` | 필드가 존재하면 필수 | `validate:"required_with=Email"` |
| `required_without=Field` | 필드가 없으면 필수 | `validate:"required_without=Phone"` |

## 지원 파일 형식

| 형식 | 확장자 | 압축 형식 |
|------|--------|----------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst` |
| Parquet | `.parquet` | `.parquet.gz`, `.parquet.bz2`, `.parquet.xz`, `.parquet.zst` |

**Parquet 압축 참고**: 외부 압축(`.parquet.gz` 등)은 컨테이너 파일 자체의 압축입니다. Parquet 파일은 내부 압축(Snappy, GZIP, LZ4, ZSTD)도 사용할 수 있으며, parquet-go 라이브러리에 의해 투명하게 처리됩니다.

**Excel 파일 참고**: **첫 번째 시트**만 처리됩니다. 여러 시트가 있는 워크북에서는 이후 시트가 무시됩니다.

## filesql과의 통합

```go
// 전처리 및 검증으로 파일 처리
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// 검증 에러 확인
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("행 %d, 컬럼 %s: %s", e.Row, e.Column, e.Message)
    }
}

// Builder 패턴을 사용하여 전처리된 데이터를 filesql에 전달
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

// 전처리된 데이터에 대해 SQL 쿼리 실행
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## 설계 고려 사항

### 이름 기반 컬럼 바인딩

struct 필드는 **이름으로** 파일 컬럼에 매핑되며, 위치가 아닙니다. 필드 이름은 자동으로 `snake_case`로 변환되어 CSV 컬럼 헤더와 매칭됩니다:

```go
// 파일 컬럼: user_name, email_address, phone_number (어떤 순서든)
type User struct {
    UserName     string  // → "user_name" 컬럼에 매칭
    EmailAddress string  // → "email_address" 컬럼에 매칭
    PhoneNumber  string  // → "phone_number" 컬럼에 매칭
}
```

**컬럼 순서는 중요하지 않습니다** - 필드는 이름으로 매칭되므로 struct를 변경하지 않고도 CSV의 컬럼 순서를 변경할 수 있습니다.

#### `name` 태그를 사용한 사용자 정의 컬럼 이름

`name` 태그를 사용하여 자동 생성된 컬럼 이름을 재정의할 수 있습니다:

```go
type User struct {
    UserName string `name:"user"`       // → "user" 컬럼에 매칭 ("user_name"이 아님)
    Email    string `name:"mail_addr"`  // → "mail_addr" 컬럼에 매칭 ("email"이 아님)
    Age      string                     // → "age" 컬럼에 매칭 (자동 snake_case)
}
```

#### 누락된 컬럼 동작

struct 필드에 대한 CSV 컬럼이 존재하지 않으면 필드 값은 빈 문자열로 처리됩니다. 검증은 여전히 실행되므로 `required`는 누락된 컬럼을 감지할 수 있습니다:

```go
type User struct {
    Name    string `validate:"required"`  // "name" 컬럼이 없으면 에러
    Country string                        // "country" 컬럼이 없으면 빈 문자열
}
```

#### 대소문자 구분 및 중복 헤더

**헤더 매칭은 대소문자를 구분하며 정확히 일치해야 합니다.** struct 필드 `UserName`은 `user_name`에 매핑되지만 `User_Name`, `USER_NAME` 또는 `userName`과 같은 헤더는 **매칭되지 않습니다**:

```go
type User struct {
    UserName string  // ✓ "user_name"에 매칭
                     // ✗ "User_Name", "USER_NAME", "userName"에는 매칭되지 않음
}
```

이는 모든 파일 형식에 적용됩니다: CSV, TSV, LTSV 키 및 Parquet/XLSX 컬럼 이름은 정확히 일치해야 합니다.

**중복 컬럼 이름:** 파일에 중복 헤더 이름(예: `id,id,name`)이 포함된 경우 **첫 번째 항목**이 바인딩에 사용됩니다:

```csv
id,id,name
first,second,John  → struct.ID = "first" (첫 번째 "id" 컬럼이 우선)
```

#### 형식별 참고 사항

**LTSV, Parquet 및 XLSX**는 동일한 대소문자 구분 매칭 규칙을 따릅니다. 키/컬럼 이름은 정확히 일치해야 합니다:

```go
type Record struct {
    UserID string                 // "user_id" 키/컬럼 기대
    Email  string `name:"EMAIL"`  // 비 snake_case 컬럼에는 name 태그 사용
}
```

LTSV 키가 하이픈(`user-id`)을 사용하거나 Parquet/XLSX 컬럼이 camelCase(`userId`)를 사용하는 경우 `name` 태그를 사용하여 정확한 컬럼 이름을 지정하세요.

### 메모리 사용량

fileprep은 처리를 위해 **전체 파일을 메모리에 로드**합니다. 이를 통해 랜덤 액세스와 다중 패스 연산이 가능하지만, 대용량 파일에는 영향이 있습니다:

| 파일 크기 | 예상 메모리 | 권장 사항 |
|-----------|-------------|-----------|
| < 100 MB | 파일 크기의 약 2-3배 | 직접 처리 |
| 100-500 MB | 500 MB - 1.5 GB | 메모리 모니터링, 청크 처리 고려 |
| > 500 MB | > 1.5 GB | 파일 분할 또는 스트리밍 대안 사용 |

압축 입력(gzip, bzip2, xz, zstd)의 경우 메모리 사용량은 **압축 해제 후** 크기를 기준으로 합니다.

## 성능

21개 컬럼을 가진 복잡한 struct로 CSV 파일을 처리하는 벤치마크 결과입니다. 각 필드는 여러 전처리 및 검증 태그를 사용합니다:

**사용된 전처리 태그:** trim, lowercase, uppercase, keep_digits, pad_left, strip_html, strip_newline, collapse_space, truncate, fix_scheme, default

**사용된 검증 태그:** required, alpha, numeric, email, uuid, ip_addr, url, oneof, min, max, len, printascii, ascii, eqfield

| 레코드 수 | 시간 | 메모리 | Allocs/op |
|--------:|-----:|-------:|----------:|
| 100 | 0.6 ms | 0.9 MB | 7,654 |
| 1,000 | 6.1 ms | 9.6 MB | 74,829 |
| 10,000 | 69 ms | 101 MB | 746,266 |
| 50,000 | 344 ms | 498 MB | 3,690,281 |

```bash
# 빠른 벤치마크 (100 및 1,000 레코드)
make bench

# 전체 벤치마크 (50,000 레코드를 포함한 모든 크기)
make bench-all
```

*AMD Ryzen AI MAX+ 395, Go 1.24, Linux에서 테스트. 결과는 하드웨어에 따라 다릅니다.*

## 관련 또는 영감을 받은 프로젝트

- [nao1215/filesql](https://github.com/nao1215/filesql) - CSV, TSV, LTSV, Parquet, Excel용 SQL 드라이버. gzip, bzip2, xz, zstd 지원.
- [nao1215/csv](https://github.com/nao1215/csv) - 검증 기능이 있는 CSV 읽기 및 간단한 DataFrame in golang.
- [go-playground/validator](https://github.com/go-playground/validator) - Go Struct 및 Field 검증, Cross Field, Cross Struct, Map, Slice, Array diving 포함.
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - 헤더가 있는 csv의 인코더/디코더.

## 기여

기여를 환영합니다! 자세한 내용은 [Contributing Guide](../../CONTRIBUTING.md)를 참조하세요.

## 지원

이 프로젝트가 유용하다면 다음을 고려해 주세요:

- GitHub에서 스타 주기 - 다른 사람들이 프로젝트를 발견하는 데 도움이 됩니다
- [스폰서 되기](https://github.com/sponsors/nao1215) - 여러분의 지원이 프로젝트를 유지하고 지속적인 개발의 동기가 됩니다

스타, 스폰서십 또는 기여를 통한 여러분의 지원이 이 프로젝트를 앞으로 나아가게 합니다. 감사합니다!

## 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다 - 자세한 내용은 [LICENSE](../../LICENSE) 파일을 참조하세요.
