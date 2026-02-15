# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[English](../../README.md) | [Español](../es/README.md) | [Français](../fr/README.md) | [한국어](../ko/README.md) | [Русский](../ru/README.md) | [中文](../zh-cn/README.md)

![fileprep-logo](../images/fileprep-logo-small.png)

**fileprep** は、CSV、TSV、LTSV、JSON、JSONL、Parquet、Excelなどの構造化データを、軽量なstructタグルールでクリーニング、正規化、バリデーションするためのGoライブラリです。gzip、bzip2、xz、zstd、zlib、snappy、s2、lz4ストリームをシームレスにサポートしています。

## なぜfileprepなのか？

私は [nao1215/filesql](https://github.com/nao1215/filesql) を開発しました。これはCSV、TSV、LTSV、Parquet、ExcelファイルにSQLクエリを実行できるライブラリです。また、CSVファイルのバリデーション用に [nao1215/csv](https://github.com/nao1215/csv) も作成しました。

機械学習を勉強する中で、「[nao1215/csv](https://github.com/nao1215/csv) を [nao1215/filesql](https://github.com/nao1215/filesql) と同じファイル形式に対応させれば、両者を組み合わせてETLのような操作ができる」と気づきました。このアイデアが **fileprep** の誕生につながりました。データの前処理/バリデーションとSQLベースのファイルクエリを橋渡しするライブラリです。

## 機能

- 複数ファイル形式対応: CSV, TSV, LTSV, JSON (.json), JSONL (.jsonl), Parquet, Excel (.xlsx)
- 圧縮対応: gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst), zlib (.z), snappy (.snappy), s2 (.s2), lz4 (.lz4)
- 名前ベースのカラムバインディング: フィールドは自動的に `snake_case` カラム名にマッチ、`name` タグでカスタマイズ可能
- structタグベースの前処理 (`prep` タグ): trim、lowercase、uppercase、デフォルト値など
- structタグベースのバリデーション (`validate` タグ): required など
- [filesql](https://github.com/nao1215/filesql) とのシームレスな統合: filesqlで直接使用できる `io.Reader` を返却
- 詳細なエラーレポート: 各エラーの行と列の情報

## インストール

```bash
go get github.com/nao1215/fileprep
```

## 必要要件

- Go バージョン: 1.24 以降
- 対応OS:
  - Linux
  - macOS
  - Windows


## クイックスタート

```go
package main

import (
    "fmt"
    "os"
    "strings"

    "github.com/nao1215/fileprep"
)

// User は前処理とバリデーション付きのユーザーレコードを表します
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

    fmt.Printf("処理完了: %d行中 %d行が有効\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("Name: %q, Email: %q\n", user.Name, user.Email)
    }

    // readerはfilesqlに直接渡すことができます
    _ = reader
}
```

出力:
```
処理完了: 2行中 2行が有効
Name: "John Doe", Email: "john@example.com"
Name: "Jane Smith", Email: "jane@example.com"
```

## 高度な使用例

### 複雑なデータの前処理とバリデーション

この例では、fileprep の全機能を示します：複数の前処理とバリデータを組み合わせて、現実世界の乱雑なデータをクリーニング・検証します。

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee は包括的な前処理とバリデーションを持つ従業員データを表します
type Employee struct {
    // ID: 6桁にパディング、数値である必要
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // 名前: 空白をクリーンアップ、必須でアルファベットとスペース
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // メール: 小文字に正規化、フォーマット検証
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // 部署: 大文字に正規化、許可された値のいずれか
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // 給与: 数字のみを抽出、範囲を検証
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // 電話番号: 数字を抽出、国コードを追加後E.164形式を検証
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // 開始日: datetime形式を検証
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // マネージャーID: 部署がHRでない場合のみ必須
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // ウェブサイト: 欠けているスキームを修正、URL検証
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // 乱雑な現実世界のCSVデータ
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
        fmt.Printf("致命的なエラー: %v\n", err)
        return
    }

    fmt.Printf("=== 処理結果 ===\n")
    fmt.Printf("総行数: %d, 有効行数: %d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("従業員 %d:\n", i+1)
        fmt.Printf("  ID:         %s\n", emp.EmployeeID)
        fmt.Printf("  名前:       %s\n", emp.FullName)
        fmt.Printf("  メール:     %s\n", emp.Email)
        fmt.Printf("  部署:       %s\n", emp.Department)
        fmt.Printf("  給与:       %s\n", emp.Salary)
        fmt.Printf("  電話番号:   %s\n", emp.Phone)
        fmt.Printf("  開始日:     %s\n", emp.StartDate)
        fmt.Printf("  マネージャーID: %s\n", emp.ManagerID)
        fmt.Printf("  ウェブサイト:   %s\n\n", emp.Website)
    }
}
```

出力:
```
=== 処理結果 ===
総行数: 4, 有効行数: 4

従業員 1:
  ID:         000042
  名前:       John Doe
  メール:     john.doe@company.com
  部署:       ENGINEERING
  給与:       75000
  電話番号:   +15551234567
  開始日:     2023-01-15
  マネージャーID: 000001
  ウェブサイト:   https://company.com/john

従業員 2:
  ID:         000007
  名前:       Jane Smith
  メール:     jane@company.com
  部署:       SALES
  給与:       120000
  電話番号:   +15559876543
  開始日:     2022-06-01
  マネージャーID: 000002
  ウェブサイト:   https://www.linkedin.com/in/jane

従業員 3:
  ID:         000123
  名前:       Bob Wilson
  メール:     bob.wilson@company.com
  部署:       HR
  給与:       45000
  電話番号:   +15551112222
  開始日:     2024-03-20
  マネージャーID: 000000
  ウェブサイト:

従業員 4:
  ID:         000099
  名前:       Alice Brown
  メール:     alice@company.com
  部署:       MARKETING
  給与:       88500
  電話番号:   +15554443333
  開始日:     2023-09-10
  マネージャーID: 000003
  ウェブサイト:   https://alice.dev
```


### 詳細なエラーレポート

バリデーションが失敗すると、fileprep は行番号、カラム名、具体的なバリデーション失敗理由を含む正確なエラー情報を提供します。

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order は厳格なバリデーションルールを持つ注文を表します
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
    // 複数のバリデーションエラーを含むCSV
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
        fmt.Printf("致命的なエラー: %v\n", err)
        return
    }

    fmt.Printf("=== バリデーションレポート ===\n")
    fmt.Printf("総行数:     %d\n", result.RowCount)
    fmt.Printf("有効行数:   %d\n", result.ValidRowCount)
    fmt.Printf("無効行数:   %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("総エラー数: %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== エラー詳細 ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("行 %d, カラム '%s': %s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

出力:
```
=== バリデーションレポート ===
総行数:     4
有効行数:   1
無効行数:   3
総エラー数: 23

=== エラー詳細 ===
行 2, カラム 'order_id': value must be a valid UUID version 4
行 2, カラム 'customer_id': value must be numeric
行 2, カラム 'email': value must be a valid email address
行 2, カラム 'amount': value must be greater than 0
行 2, カラム 'currency': value must have exactly 3 characters
行 2, カラム 'country': value must have exactly 2 characters
行 2, カラム 'order_date': value must be a valid datetime in format: 2006-01-02
行 2, カラム 'ip_address': value must be a valid IP address
行 2, カラム 'promo_code': value must contain only alphanumeric characters
行 2, カラム 'quantity': value must be greater than or equal to 1
行 2, カラム 'unit_price': value must be greater than 0
行 2, カラム 'ship_date': value must be greater than field OrderDate
行 2, カラム 'total_check': value must equal field Amount
行 3, カラム 'customer_id': value is required
行 3, カラム 'email': value must be a valid email address
行 3, カラム 'amount': value must be less than or equal to 10000
行 3, カラム 'currency': value must have exactly 3 characters
行 3, カラム 'country': value must contain only alphabetic characters
行 3, カラム 'order_date': value must be a valid datetime in format: 2006-01-02
行 3, カラム 'quantity': value must be less than or equal to 100
行 3, カラム 'unit_price': value must be greater than 0
行 3, カラム 'ship_date': value must be greater than field OrderDate
行 4, カラム 'ship_date': value must be greater than field OrderDate
```

## 前処理タグ (`prep`)

複数のタグを組み合わせることができます: `prep:"trim,lowercase,default=N/A"`

### 基本的な前処理

| タグ | 説明 | 例 |
|-----|------|-----|
| `trim` | 前後の空白を削除 | `prep:"trim"` |
| `ltrim` | 先頭の空白を削除 | `prep:"ltrim"` |
| `rtrim` | 末尾の空白を削除 | `prep:"rtrim"` |
| `lowercase` | 小文字に変換 | `prep:"lowercase"` |
| `uppercase` | 大文字に変換 | `prep:"uppercase"` |
| `default=value` | 空の場合にデフォルト値を設定 | `prep:"default=N/A"` |

### 文字列変換

| タグ | 説明 | 例 |
|-----|------|-----|
| `replace=old:new` | すべての出現を置換 | `prep:"replace=;:,"` |
| `prefix=value` | 文字列を先頭に追加 | `prep:"prefix=ID_"` |
| `suffix=value` | 文字列を末尾に追加 | `prep:"suffix=_END"` |
| `truncate=N` | N文字に制限 | `prep:"truncate=100"` |
| `strip_html` | HTMLタグを削除 | `prep:"strip_html"` |
| `strip_newline` | 改行を削除 (LF, CRLF, CR) | `prep:"strip_newline"` |
| `collapse_space` | 複数のスペースを1つに | `prep:"collapse_space"` |

### 文字フィルタリング

| タグ | 説明 | 例 |
|-----|------|-----|
| `remove_digits` | すべての数字を削除 | `prep:"remove_digits"` |
| `remove_alpha` | すべてのアルファベットを削除 | `prep:"remove_alpha"` |
| `keep_digits` | 数字のみを保持 | `prep:"keep_digits"` |
| `keep_alpha` | アルファベットのみを保持 | `prep:"keep_alpha"` |
| `trim_set=chars` | 指定文字を両端から削除 | `prep:"trim_set=@#$"` |

### パディング

| タグ | 説明 | 例 |
|-----|------|-----|
| `pad_left=N:char` | N文字まで左にパディング | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | N文字まで右にパディング | `prep:"pad_right=10: "` |

### 高度な前処理

| タグ | 説明 | 例 |
|-----|------|-----|
| `normalize_unicode` | UnicodeをNFC形式に正規化 | `prep:"normalize_unicode"` |
| `nullify=value` | 特定の文字列を空として扱う | `prep:"nullify=NULL"` |
| `coerce=type` | 型変換 (int, float, bool) | `prep:"coerce=int"` |
| `fix_scheme=scheme` | URLスキームを追加/修正 | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | 正規表現による置換 | `prep:"regex_replace=\\d+:X"` |

## バリデーションタグ (`validate`)

複数のタグを組み合わせることができます: `validate:"required,email"`

### 基本的なバリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `required` | フィールドは空であってはならない | `validate:"required"` |
| `boolean` | true, false, 0, または 1 である必要がある | `validate:"boolean"` |

### 文字タイプバリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `alpha` | ASCIIアルファベット文字のみ | `validate:"alpha"` |
| `alphaunicode` | Unicodeレター文字のみ | `validate:"alphaunicode"` |
| `alphaspace` | アルファベット文字またはスペース | `validate:"alphaspace"` |
| `alphanumeric` | ASCII英数字 | `validate:"alphanumeric"` |
| `alphanumunicode` | Unicodeレターまたは数字 | `validate:"alphanumunicode"` |
| `numeric` | 有効な整数 | `validate:"numeric"` |
| `number` | 有効な数値（整数または小数） | `validate:"number"` |
| `ascii` | ASCII文字のみ | `validate:"ascii"` |
| `printascii` | 印刷可能ASCII文字 (0x20-0x7E) | `validate:"printascii"` |
| `multibyte` | マルチバイト文字を含む | `validate:"multibyte"` |

### 数値比較バリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `eq=N` | 値がNと等しい | `validate:"eq=100"` |
| `ne=N` | 値がNと等しくない | `validate:"ne=0"` |
| `gt=N` | 値がNより大きい | `validate:"gt=0"` |
| `gte=N` | 値がN以上 | `validate:"gte=1"` |
| `lt=N` | 値がN未満 | `validate:"lt=100"` |
| `lte=N` | 値がN以下 | `validate:"lte=99"` |
| `min=N` | 値が最低N | `validate:"min=0"` |
| `max=N` | 値が最大N | `validate:"max=100"` |
| `len=N` | 正確にN文字 | `validate:"len=10"` |

### 文字列バリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `oneof=a b c` | 許可された値のいずれか | `validate:"oneof=active inactive"` |
| `lowercase` | すべて小文字 | `validate:"lowercase"` |
| `uppercase` | すべて大文字 | `validate:"uppercase"` |
| `eq_ignore_case=value` | 大文字小文字を無視した等価 | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | 大文字小文字を無視した非等価 | `validate:"ne_ignore_case=no"` |

### 文字列内容バリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `startswith=prefix` | 接頭辞で始まる | `validate:"startswith=http"` |
| `startsnotwith=prefix` | 接頭辞で始まらない | `validate:"startsnotwith=_"` |
| `endswith=suffix` | 接尾辞で終わる | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | 接尾辞で終わらない | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | 部分文字列を含む | `validate:"contains=@"` |
| `containsany=chars` | 指定文字のいずれかを含む | `validate:"containsany=abc"` |
| `containsrune=r` | 指定ルーンを含む | `validate:"containsrune=@"` |
| `excludes=substr` | 部分文字列を含まない | `validate:"excludes=admin"` |
| `excludesall=chars` | 指定文字のいずれも含まない | `validate:"excludesall=<>"` |
| `excludesrune=r` | 指定ルーンを含まない | `validate:"excludesrune=$"` |

### フォーマットバリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `email` | 有効なメールアドレス | `validate:"email"` |
| `uri` | 有効なURI | `validate:"uri"` |
| `url` | 有効なURL | `validate:"url"` |
| `http_url` | 有効なHTTPまたはHTTPS URL | `validate:"http_url"` |
| `https_url` | 有効なHTTPS URL | `validate:"https_url"` |
| `url_encoded` | URLエンコードされた文字列 | `validate:"url_encoded"` |
| `datauri` | 有効なデータURI | `validate:"datauri"` |
| `datetime=layout` | Goレイアウトに一致する有効な日時 | `validate:"datetime=2006-01-02"` |
| `uuid` | 有効なUUID（任意のバージョン） | `validate:"uuid"` |
| `uuid3` | 有効なUUIDバージョン3 | `validate:"uuid3"` |
| `uuid4` | 有効なUUIDバージョン4 | `validate:"uuid4"` |
| `uuid5` | 有効なUUIDバージョン5 | `validate:"uuid5"` |
| `ulid` | 有効なULID | `validate:"ulid"` |
| `e164` | 有効なE.164電話番号 | `validate:"e164"` |
| `latitude` | 有効な緯度 (-90 to 90) | `validate:"latitude"` |
| `longitude` | 有効な経度 (-180 to 180) | `validate:"longitude"` |
| `hexadecimal` | 有効な16進数文字列 | `validate:"hexadecimal"` |
| `hexcolor` | 有効な16進数カラーコード | `validate:"hexcolor"` |
| `rgb` | 有効なRGBカラー | `validate:"rgb"` |
| `rgba` | 有効なRGBAカラー | `validate:"rgba"` |
| `hsl` | 有効なHSLカラー | `validate:"hsl"` |
| `hsla` | 有効なHSLAカラー | `validate:"hsla"` |

### ネットワークバリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `ip_addr` | 有効なIPアドレス (v4またはv6) | `validate:"ip_addr"` |
| `ip4_addr` | 有効なIPv4アドレス | `validate:"ip4_addr"` |
| `ip6_addr` | 有効なIPv6アドレス | `validate:"ip6_addr"` |
| `cidr` | 有効なCIDR表記 | `validate:"cidr"` |
| `cidrv4` | 有効なIPv4 CIDR | `validate:"cidrv4"` |
| `cidrv6` | 有効なIPv6 CIDR | `validate:"cidrv6"` |
| `mac` | 有効なMACアドレス | `validate:"mac"` |
| `fqdn` | 有効な完全修飾ドメイン名 | `validate:"fqdn"` |
| `hostname` | 有効なホスト名 (RFC 952) | `validate:"hostname"` |
| `hostname_rfc1123` | 有効なホスト名 (RFC 1123) | `validate:"hostname_rfc1123"` |
| `hostname_port` | 有効なホスト名:ポート | `validate:"hostname_port"` |

### クロスフィールドバリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `eqfield=Field` | 値が別のフィールドと等しい | `validate:"eqfield=Password"` |
| `nefield=Field` | 値が別のフィールドと等しくない | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | 値が別のフィールドより大きい | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | 値が別のフィールド以上 | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | 値が別のフィールドより小さい | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | 値が別のフィールド以下 | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | 別のフィールドの値を含む | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | 別のフィールドの値を含まない | `validate:"fieldexcludes=Forbidden"` |

### 条件付き必須バリデータ

| タグ | 説明 | 例 |
|-----|------|-----|
| `required_if=Field value` | フィールドが値と等しい場合は必須 | `validate:"required_if=Status active"` |
| `required_unless=Field value` | フィールドが値と等しくない場合は必須 | `validate:"required_unless=Type guest"` |
| `required_with=Field` | フィールドが存在する場合は必須 | `validate:"required_with=Email"` |
| `required_without=Field` | フィールドが存在しない場合は必須 | `validate:"required_without=Phone"` |

## サポートされているファイル形式

| 形式 | 拡張子 | 圧縮形式 |
|------|--------|----------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst`, `.csv.z`, `.csv.snappy`, `.csv.s2`, `.csv.lz4` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst`, `.tsv.z`, `.tsv.snappy`, `.tsv.s2`, `.tsv.lz4` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst`, `.ltsv.z`, `.ltsv.snappy`, `.ltsv.s2`, `.ltsv.lz4` |
| JSON | `.json` | `.json.gz`, `.json.bz2`, `.json.xz`, `.json.zst`, `.json.z`, `.json.snappy`, `.json.s2`, `.json.lz4` |
| JSONL | `.jsonl` | `.jsonl.gz`, `.jsonl.bz2`, `.jsonl.xz`, `.jsonl.zst`, `.jsonl.z`, `.jsonl.snappy`, `.jsonl.s2`, `.jsonl.lz4` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst`, `.xlsx.z`, `.xlsx.snappy`, `.xlsx.s2`, `.xlsx.lz4` |
| Parquet | `.parquet` | `.parquet.gz`, `.parquet.bz2`, `.parquet.xz`, `.parquet.zst`, `.parquet.z`, `.parquet.snappy`, `.parquet.s2`, `.parquet.lz4` |

### サポートされている圧縮形式

| 形式 | 拡張子 | ライブラリ | 備考 |
|------|--------|-----------|------|
| gzip | `.gz` | compress/gzip | 標準ライブラリ |
| bzip2 | `.bz2` | compress/bzip2 | 標準ライブラリ |
| xz | `.xz` | github.com/ulikunitz/xz | Pure Go |
| zstd | `.zst` | github.com/klauspost/compress/zstd | Pure Go、高性能 |
| zlib | `.z` | compress/zlib | 標準ライブラリ |
| snappy | `.snappy` | github.com/klauspost/compress/snappy | Pure Go、高性能 |
| s2 | `.s2` | github.com/klauspost/compress/s2 | Snappy互換、より高速 |
| lz4 | `.lz4` | github.com/pierrec/lz4/v4 | Pure Go |

**Parquet圧縮についての注意**: 外部圧縮（`.parquet.gz`など）はコンテナファイル自体の圧縮です。Parquetファイルは内部圧縮（Snappy、GZIP、LZ4、ZSTD）も使用でき、parquet-goライブラリによって透過的に処理されます。

**Excelファイルについての注意**: **最初のシート**のみが処理されます。複数シートのワークブックでは、後続のシートは無視されます。

**JSON/JSONLファイルについての注意**: JSON/JSONLデータは生のJSON文字列を含む単一の `"data"` カラムに格納されます。JSON配列の各要素やJSONLの各行が1行になります。JSON入力はコンパクトなJSONL（1行に1つのJSON値）として出力されます。前処理タグは生のJSON文字列に対して動作し、内部の個別フィールドには作用しません。前処理がJSON構造を破壊した場合、`Process` は `ErrInvalidJSONAfterPrep` を返します。前処理後にすべての行が空になった場合、`Process` は `ErrEmptyJSONOutput` を返します。

## filesqlとの連携

```go
// 前処理とバリデーションでファイルを処理
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// バリデーションエラーをチェック
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("行 %d, カラム %s: %s", e.Row, e.Column, e.Message)
    }
}

// Builderパターンを使用して前処理済みデータをfilesqlに渡す
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

// 前処理済みデータに対してSQLクエリを実行
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## 設計上の考慮事項

### 名前ベースのカラムバインディング

structフィールドはファイルカラムに**名前で**マッピングされ、位置ではありません。フィールド名は自動的に `snake_case` に変換されてCSVカラムヘッダーにマッチします：

```go
// ファイルカラム: user_name, email_address, phone_number (任意の順序)
type User struct {
    UserName     string  // → "user_name" カラムにマッチ
    EmailAddress string  // → "email_address" カラムにマッチ
    PhoneNumber  string  // → "phone_number" カラムにマッチ
}
```

**カラムの順序は関係ありません** - フィールドは名前でマッチされるため、structを変更せずにCSVのカラム順序を変更できます。

#### `name` タグによるカスタムカラム名

`name` タグを使用して自動生成されたカラム名を上書きできます：

```go
type User struct {
    UserName string `name:"user"`       // → "user" カラムにマッチ ("user_name" ではなく)
    Email    string `name:"mail_addr"`  // → "mail_addr" カラムにマッチ ("email" ではなく)
    Age      string                     // → "age" カラムにマッチ (自動snake_case)
}
```

#### 欠落カラムの動作

structフィールドに対応するCSVカラムが存在しない場合、フィールド値は空文字列として扱われます。バリデーションは引き続き実行されるため、`required` は欠落カラムを検出できます：

```go
type User struct {
    Name    string `validate:"required"`  // "name" カラムがない場合はエラー
    Country string                        // "country" カラムがない場合は空文字列
}
```

#### 大文字小文字の区別と重複ヘッダー

**ヘッダーマッチングは大文字小文字を区別し、完全一致です。** structフィールド `UserName` は `user_name` にマッピングされますが、`User_Name`、`USER_NAME`、または `userName` のようなヘッダーは**マッチしません**：

```go
type User struct {
    UserName string  // ✓ "user_name" にマッチ
                     // ✗ "User_Name", "USER_NAME", "userName" にはマッチしない
}
```

これはすべてのファイル形式に適用されます：CSV、TSV、LTSVキー、およびParquet/XLSXカラム名は正確に一致する必要があります。

**重複カラム名:** ファイルに重複するヘッダー名（例：`id,id,name`）が含まれている場合、**最初の出現**がバインディングに使用されます：

```csv
id,id,name
first,second,John  → struct.ID = "first" (最初の "id" カラムが優先)
```

#### 形式固有の注意事項

**LTSV、Parquet、およびXLSX** は同じ大文字小文字を区別するマッチングルールに従います。キー/カラム名は正確に一致する必要があります：

```go
type Record struct {
    UserID string                 // "user_id" キー/カラムを期待
    Email  string `name:"EMAIL"`  // 非snake_caseカラムにはnameタグを使用
}
```

LTSVキーがハイフン（`user-id`）を使用している場合や、Parquet/XLSXカラムがcamelCase（`userId`）を使用している場合は、`name` タグを使用して正確なカラム名を指定してください。

### メモリ使用量

fileprepは処理のために**ファイル全体をメモリに読み込みます**。これによりランダムアクセスとマルチパス操作が可能になりますが、大きなファイルには影響があります：

| ファイルサイズ | 概算メモリ | 推奨事項 |
|----------------|------------|----------|
| < 100 MB | ファイルサイズの約2-3倍 | 直接処理 |
| 100-500 MB | 500 MB - 1.5 GB | メモリ監視、チャンク処理を検討 |
| > 500 MB | > 1.5 GB | ファイル分割またはストリーミング代替案を使用 |

圧縮入力（gzip、bzip2、xz、zstd、zlib、snappy、s2、lz4）の場合、メモリ使用量は**解凍後**のサイズに基づきます。

## パフォーマンス

21カラムを持つ複雑なstructでCSVファイルを処理するベンチマーク結果。各フィールドは複数の前処理タグとバリデーションタグを使用しています：

**使用した前処理タグ:** trim, lowercase, uppercase, keep_digits, pad_left, strip_html, strip_newline, collapse_space, truncate, fix_scheme, default

**使用したバリデーションタグ:** required, alpha, numeric, email, uuid, ip_addr, url, oneof, min, max, len, printascii, ascii, eqfield

| レコード数 | 時間 | メモリ | Allocs/op |
|--------:|-----:|-------:|----------:|
| 100 | 0.6 ms | 0.9 MB | 7,654 |
| 1,000 | 6.1 ms | 9.6 MB | 74,829 |
| 10,000 | 69 ms | 101 MB | 746,266 |
| 50,000 | 344 ms | 498 MB | 3,690,281 |

```bash
# クイックベンチマーク（100および1,000レコード）
make bench

# フルベンチマーク（50,000レコードを含むすべてのサイズ）
make bench-all
```

*AMD Ryzen AI MAX+ 395、Go 1.24、Linuxでテスト。結果はハードウェアにより異なります。*

## 関連・インスパイアされたプロジェクト

- [nao1215/filesql](https://github.com/nao1215/filesql) - CSV、TSV、LTSV、Parquet、Excel用のSQLドライバー。gzip、bzip2、xz、zstdをサポート。
- [nao1215/csv](https://github.com/nao1215/csv) - バリデーション付きCSV読み込みとシンプルなDataFrame。
- [go-playground/validator](https://github.com/go-playground/validator) - Go Structおよびフィールドバリデーション、クロスフィールド、クロスStruct、Map、Slice、Arrayの探索を含む。
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - ヘッダー付きCSVのエンコーダー/デコーダー。

## コントリビューション

コントリビューションを歓迎します！詳細は [Contributing Guide](../../CONTRIBUTING.md) をご覧ください。

## サポート

このプロジェクトが役に立つと思われた場合は、以下をご検討ください：

- GitHubでスターを付ける - 他の人がプロジェクトを発見するのに役立ちます
- [スポンサーになる](https://github.com/sponsors/nao1215) - あなたのサポートがプロジェクトを存続させ、継続的な開発のモチベーションになります

スター、スポンサーシップ、コントリビューションを通じたあなたのサポートが、このプロジェクトを前進させる力です。ありがとうございます！

## ライセンス

このプロジェクトはMITライセンスの下でライセンスされています - 詳細は [LICENSE](../../LICENSE) ファイルをご覧ください。
