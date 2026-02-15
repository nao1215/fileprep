# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[English](../../README.md) | [日本語](../ja/README.md) | [Español](../es/README.md) | [Français](../fr/README.md) | [한국어](../ko/README.md) | [Русский](../ru/README.md)

![fileprep-logo](../images/fileprep-logo-small.png)

**fileprep** 是一个用于清理、规范化和验证结构化数据（CSV、TSV、LTSV、JSON、JSONL、Parquet 和 Excel）的 Go 库，通过轻量级的 struct 标签规则实现，无缝支持 gzip、bzip2、xz、zstd、zlib、snappy、s2 和 lz4 压缩流。

## 为什么选择 fileprep？

我开发了 [nao1215/filesql](https://github.com/nao1215/filesql)，它可以对 CSV、TSV、LTSV、Parquet 和 Excel 文件执行 SQL 查询。我还创建了 [nao1215/csv](https://github.com/nao1215/csv) 用于 CSV 文件验证。

在学习机器学习的过程中，我意识到："如果将 [nao1215/csv](https://github.com/nao1215/csv) 扩展为支持与 [nao1215/filesql](https://github.com/nao1215/filesql) 相同的文件格式，就可以将两者结合起来执行类似 ETL 的操作"。这个想法促成了 **fileprep** 的诞生：一个连接数据预处理/验证与基于 SQL 的文件查询的库。

## 功能

- 多文件格式支持：CSV、TSV、LTSV、JSON (.json)、JSONL (.jsonl)、Parquet、Excel (.xlsx)
- 压缩支持：gzip (.gz)、bzip2 (.bz2)、xz (.xz)、zstd (.zst)、zlib (.z)、snappy (.snappy)、s2 (.s2)、lz4 (.lz4)
- 基于名称的列绑定：字段自动匹配 `snake_case` 列名，可通过 `name` 标签自定义
- 基于 struct 标签的预处理（`prep` 标签）：trim、lowercase、uppercase、默认值等
- 基于 struct 标签的验证（`validate` 标签）：required、omitempty 等
- [filesql](https://github.com/nao1215/filesql) 无缝集成：返回 `io.Reader` 可直接用于 filesql
- 处理器选项：`WithStrictTagParsing()` 用于检测标签配置错误，`WithValidRowsOnly()` 用于过滤输出
- 详细的错误报告：每个错误包含行和列信息

## 安装

```bash
go get github.com/nao1215/fileprep
```

## 要求

- Go 版本：1.24 或更高
- 支持的操作系统：
  - Linux
  - macOS
  - Windows


## 快速开始

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// User 表示一个带有预处理和验证的用户记录
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
        fmt.Printf("错误：%v\n", err)
        return
    }

    fmt.Printf("处理完成：%d 行中 %d 行有效\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("姓名：%q，邮箱：%q\n", user.Name, user.Email)
    }

    // reader 可以直接传递给 filesql
    _ = reader
}
```

输出：
```
处理完成：2 行中 2 行有效
姓名："John Doe"，邮箱："john@example.com"
姓名："Jane Smith"，邮箱："jane@example.com"
```

## 使用 fileprep 之前

### JSON/JSONL 使用单个 "data" 列

JSON/JSONL 文件被解析为名为 `"data"` 的单列。每个数组元素（JSON）或每一行（JSONL）成为包含原始 JSON 字符串的一行。

```go
type JSONRecord struct {
    Data string `name:"data" prep:"trim" validate:"required"`
}
```

输出始终是紧凑的 JSONL。如果 prep 标签破坏了 JSON 结构，`Process` 返回 `ErrInvalidJSONAfterPrep`。如果所有行最终都为空，则返回 `ErrEmptyJSONOutput`。

### 列匹配区分大小写

`UserName` 通过自动 snake_case 映射到 `user_name`。`User_Name`、`USER_NAME`、`userName` 等表头**不会**匹配。当表头不同时使用 `name` 标签：

```go
type Record struct {
    UserName string                 // 仅匹配 "user_name"
    Email    string `name:"EMAIL"`  // 精确匹配 "EMAIL"
}
```

### 重复表头：第一列优先

如果文件包含 `id,id,name`，第一个 `id` 列用于绑定，第二个被忽略。

### 缺失列变为空字符串

如果结构体字段对应的列不存在，值为 `""`。添加 `validate:"required"` 可在解析时捕获此情况。

### Excel：仅处理第一个工作表

多工作表的 `.xlsx` 文件将静默忽略第一个之后的所有工作表。

## 高级示例

### 复杂数据预处理和验证

此示例展示了 fileprep 的全部功能：组合多个预处理器和验证器来清理和验证真实的"脏"数据。

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee 表示带有综合预处理和验证的员工数据
type Employee struct {
    // ID：填充到6位数字，必须是数字
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // 姓名：清理空白，必须是字母和空格
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // 邮箱：规范化为小写，验证格式
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // 部门：规范化大小写，必须是允许的值之一
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // 薪资：只保留数字，验证范围
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // 电话：提取数字，添加国家代码后验证 E.164 格式
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // 入职日期：验证日期格式
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // 经理 ID：如果部门不是 HR 则必填
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // 网站：修复缺失的协议，验证 URL
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // 真实的"脏" CSV 数据
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
        fmt.Printf("致命错误：%v\n", err)
        return
    }

    fmt.Printf("=== 处理结果 ===\n")
    fmt.Printf("总行数：%d，有效行数：%d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("员工 %d：\n", i+1)
        fmt.Printf("  ID：       %s\n", emp.EmployeeID)
        fmt.Printf("  姓名：     %s\n", emp.FullName)
        fmt.Printf("  邮箱：     %s\n", emp.Email)
        fmt.Printf("  部门：     %s\n", emp.Department)
        fmt.Printf("  薪资：     %s\n", emp.Salary)
        fmt.Printf("  电话：     %s\n", emp.Phone)
        fmt.Printf("  入职日期： %s\n", emp.StartDate)
        fmt.Printf("  经理 ID：  %s\n", emp.ManagerID)
        fmt.Printf("  网站：     %s\n\n", emp.Website)
    }
}
```

输出：
```
=== 处理结果 ===
总行数：4，有效行数：4

员工 1：
  ID：       000042
  姓名：     John Doe
  邮箱：     john.doe@company.com
  部门：     ENGINEERING
  薪资：     75000
  电话：     +15551234567
  入职日期： 2023-01-15
  经理 ID：  000001
  网站：     https://company.com/john

员工 2：
  ID：       000007
  姓名：     Jane Smith
  邮箱：     jane@company.com
  部门：     SALES
  薪资：     120000
  电话：     +15559876543
  入职日期： 2022-06-01
  经理 ID：  000002
  网站：     https://www.linkedin.com/in/jane

员工 3：
  ID：       000123
  姓名：     Bob Wilson
  邮箱：     bob.wilson@company.com
  部门：     HR
  薪资：     45000
  电话：     +15551112222
  入职日期： 2024-03-20
  经理 ID：  000000
  网站：

员工 4：
  ID：       000099
  姓名：     Alice Brown
  邮箱：     alice@company.com
  部门：     MARKETING
  薪资：     88500
  电话：     +15554443333
  入职日期： 2023-09-10
  经理 ID：  000003
  网站：     https://alice.dev
```


### 详细错误报告

当验证失败时，fileprep 提供精确的错误信息，包括行号、列名和具体的验证失败原因。

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order 表示具有严格验证规则的订单
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
    // 包含多个验证错误的 CSV
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
        fmt.Printf("致命错误：%v\n", err)
        return
    }

    fmt.Printf("=== 验证报告 ===\n")
    fmt.Printf("总行数：   %d\n", result.RowCount)
    fmt.Printf("有效行数： %d\n", result.ValidRowCount)
    fmt.Printf("无效行数： %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("总错误数： %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== 错误详情 ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("第 %d 行，列 '%s'：%s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

输出：
```
=== 验证报告 ===
总行数：   4
有效行数： 1
无效行数： 3
总错误数： 23

=== 错误详情 ===
第 2 行，列 'order_id'：value must be a valid UUID version 4
第 2 行，列 'customer_id'：value must be numeric
第 2 行，列 'email'：value must be a valid email address
第 2 行，列 'amount'：value must be greater than 0
第 2 行，列 'currency'：value must have exactly 3 characters
第 2 行，列 'country'：value must have exactly 2 characters
第 2 行，列 'order_date'：value must be a valid datetime in format: 2006-01-02
第 2 行，列 'ip_address'：value must be a valid IP address
第 2 行，列 'promo_code'：value must contain only alphanumeric characters
第 2 行，列 'quantity'：value must be greater than or equal to 1
第 2 行，列 'unit_price'：value must be greater than 0
第 2 行，列 'ship_date'：value must be greater than field OrderDate
第 2 行，列 'total_check'：value must equal field Amount
第 3 行，列 'customer_id'：value is required
第 3 行，列 'email'：value must be a valid email address
第 3 行，列 'amount'：value must be less than or equal to 10000
第 3 行，列 'currency'：value must have exactly 3 characters
第 3 行，列 'country'：value must contain only alphabetic characters
第 3 行，列 'order_date'：value must be a valid datetime in format: 2006-01-02
第 3 行，列 'quantity'：value must be less than or equal to 100
第 3 行，列 'unit_price'：value must be greater than 0
第 3 行，列 'ship_date'：value must be greater than field OrderDate
第 4 行，列 'ship_date'：value must be greater than field OrderDate
```

## 预处理标签（`prep`）

可以组合多个标签：`prep:"trim,lowercase,default=N/A"`

### 基本预处理器

| 标签 | 描述 | 示例 |
|------|------|------|
| `trim` | 删除前后空白 | `prep:"trim"` |
| `ltrim` | 删除前导空白 | `prep:"ltrim"` |
| `rtrim` | 删除尾部空白 | `prep:"rtrim"` |
| `lowercase` | 转换为小写 | `prep:"lowercase"` |
| `uppercase` | 转换为大写 | `prep:"uppercase"` |
| `default=value` | 如果为空则设置默认值 | `prep:"default=N/A"` |

### 字符串转换

| 标签 | 描述 | 示例 |
|------|------|------|
| `replace=old:new` | 替换所有出现 | `prep:"replace=;:,"` |
| `prefix=value` | 在开头添加字符串 | `prep:"prefix=ID_"` |
| `suffix=value` | 在结尾添加字符串 | `prep:"suffix=_END"` |
| `truncate=N` | 限制为 N 个字符 | `prep:"truncate=100"` |
| `strip_html` | 删除 HTML 标签 | `prep:"strip_html"` |
| `strip_newline` | 删除换行符（LF、CRLF、CR） | `prep:"strip_newline"` |
| `collapse_space` | 将多个空格压缩为一个 | `prep:"collapse_space"` |

### 字符过滤

| 标签 | 描述 | 示例 |
|------|------|------|
| `remove_digits` | 删除所有数字 | `prep:"remove_digits"` |
| `remove_alpha` | 删除所有字母 | `prep:"remove_alpha"` |
| `keep_digits` | 只保留数字 | `prep:"keep_digits"` |
| `keep_alpha` | 只保留字母 | `prep:"keep_alpha"` |
| `trim_set=chars` | 从两端删除指定字符 | `prep:"trim_set=@#$"` |

### 填充

| 标签 | 描述 | 示例 |
|------|------|------|
| `pad_left=N:char` | 左填充至 N 个字符 | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | 右填充至 N 个字符 | `prep:"pad_right=10: "` |

### 高级预处理

| 标签 | 描述 | 示例 |
|------|------|------|
| `normalize_unicode` | 将 Unicode 规范化为 NFC 格式 | `prep:"normalize_unicode"` |
| `nullify=value` | 将特定字符串视为空 | `prep:"nullify=NULL"` |
| `coerce=type` | 类型转换（int、float、bool） | `prep:"coerce=int"` |
| `fix_scheme=scheme` | 添加/修复 URL 方案 | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | 正则表达式替换 | `prep:"regex_replace=\\d+:X"` |

## 验证标签（`validate`）

可以组合多个标签：`validate:"required,email"`

### 基本验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `required` | 字段不能为空 | `validate:"required"` |
| `omitempty` | 如果值为空则跳过后续验证器 | `validate:"omitempty,email"` |
| `boolean` | 必须是 true、false、0 或 1 | `validate:"boolean"` |

### 字符类型验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `alpha` | 仅 ASCII 字母 | `validate:"alpha"` |
| `alphaunicode` | 仅 Unicode 字母 | `validate:"alphaunicode"` |
| `alphaspace` | 字母或空格 | `validate:"alphaspace"` |
| `alphanumeric` | ASCII 字母和数字 | `validate:"alphanumeric"` |
| `alphanumunicode` | Unicode 字母或数字 | `validate:"alphanumunicode"` |
| `numeric` | 整数 | `validate:"numeric"` |
| `number` | 数字（整数或小数） | `validate:"number"` |
| `ascii` | 仅 ASCII 字符 | `validate:"ascii"` |
| `printascii` | 可打印 ASCII 字符（0x20-0x7E） | `validate:"printascii"` |
| `multibyte` | 包含多字节字符 | `validate:"multibyte"` |

### 数值比较验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `eq=N` | 值等于 N | `validate:"eq=100"` |
| `ne=N` | 值不等于 N | `validate:"ne=0"` |
| `gt=N` | 值大于 N | `validate:"gt=0"` |
| `gte=N` | 值大于或等于 N | `validate:"gte=1"` |
| `lt=N` | 值小于 N | `validate:"lt=100"` |
| `lte=N` | 值小于或等于 N | `validate:"lte=99"` |
| `min=N` | 值至少为 N | `validate:"min=0"` |
| `max=N` | 值最多为 N | `validate:"max=100"` |
| `len=N` | 恰好 N 个字符 | `validate:"len=10"` |

### 字符串验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `oneof=a b c` | 值是允许值之一 | `validate:"oneof=active inactive"` |
| `lowercase` | 值为小写 | `validate:"lowercase"` |
| `uppercase` | 值为大写 | `validate:"uppercase"` |
| `eq_ignore_case=value` | 忽略大小写相等 | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | 忽略大小写不相等 | `validate:"ne_ignore_case=no"` |

### 字符串内容验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `startswith=prefix` | 值以 prefix 开头 | `validate:"startswith=http"` |
| `startsnotwith=prefix` | 值不以 prefix 开头 | `validate:"startsnotwith=_"` |
| `endswith=suffix` | 值以 suffix 结尾 | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | 值不以 suffix 结尾 | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | 值包含子字符串 | `validate:"contains=@"` |
| `containsany=chars` | 值包含任一字符 | `validate:"containsany=abc"` |
| `containsrune=r` | 值包含该字符 | `validate:"containsrune=@"` |
| `excludes=substr` | 值不包含子字符串 | `validate:"excludes=admin"` |
| `excludesall=chars` | 值不包含任何这些字符 | `validate:"excludesall=<>"` |
| `excludesrune=r` | 值不包含该字符 | `validate:"excludesrune=$"` |

### 格式验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `email` | 有效的电子邮件地址 | `validate:"email"` |
| `uri` | 有效的 URI | `validate:"uri"` |
| `url` | 有效的 URL | `validate:"url"` |
| `http_url` | 有效的 HTTP 或 HTTPS URL | `validate:"http_url"` |
| `https_url` | 有效的 HTTPS URL | `validate:"https_url"` |
| `url_encoded` | URL 编码的字符串 | `validate:"url_encoded"` |
| `datauri` | 有效的 data URI | `validate:"datauri"` |
| `datetime=layout` | 符合 Go 格式的有效日期时间 | `validate:"datetime=2006-01-02"` |
| `uuid` | 有效的 UUID（任意版本） | `validate:"uuid"` |
| `uuid3` | 有效的 UUID 版本 3 | `validate:"uuid3"` |
| `uuid4` | 有效的 UUID 版本 4 | `validate:"uuid4"` |
| `uuid5` | 有效的 UUID 版本 5 | `validate:"uuid5"` |
| `ulid` | 有效的 ULID | `validate:"ulid"` |
| `e164` | 有效的 E.164 电话号码 | `validate:"e164"` |
| `latitude` | 有效的纬度（-90 到 90） | `validate:"latitude"` |
| `longitude` | 有效的经度（-180 到 180） | `validate:"longitude"` |
| `hexadecimal` | 有效的十六进制字符串 | `validate:"hexadecimal"` |
| `hexcolor` | 有效的十六进制颜色代码 | `validate:"hexcolor"` |
| `rgb` | 有效的 RGB 颜色 | `validate:"rgb"` |
| `rgba` | 有效的 RGBA 颜色 | `validate:"rgba"` |
| `hsl` | 有效的 HSL 颜色 | `validate:"hsl"` |
| `hsla` | 有效的 HSLA 颜色 | `validate:"hsla"` |

### 网络验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `ip_addr` | 有效的 IP 地址（v4 或 v6） | `validate:"ip_addr"` |
| `ip4_addr` | 有效的 IPv4 地址 | `validate:"ip4_addr"` |
| `ip6_addr` | 有效的 IPv6 地址 | `validate:"ip6_addr"` |
| `cidr` | 有效的 CIDR 表示法 | `validate:"cidr"` |
| `cidrv4` | 有效的 IPv4 CIDR | `validate:"cidrv4"` |
| `cidrv6` | 有效的 IPv6 CIDR | `validate:"cidrv6"` |
| `mac` | 有效的 MAC 地址 | `validate:"mac"` |
| `fqdn` | 有效的完全限定域名 | `validate:"fqdn"` |
| `hostname` | 有效的主机名（RFC 952） | `validate:"hostname"` |
| `hostname_rfc1123` | 有效的主机名（RFC 1123） | `validate:"hostname_rfc1123"` |
| `hostname_port` | 有效的 hostname:port | `validate:"hostname_port"` |

### 跨字段验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `eqfield=Field` | 值等于另一个字段 | `validate:"eqfield=Password"` |
| `nefield=Field` | 值不等于另一个字段 | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | 值大于另一个字段 | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | 值 >= 另一个字段 | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | 值小于另一个字段 | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | 值 <= 另一个字段 | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | 值包含另一个字段的值 | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | 值不包含另一个字段的值 | `validate:"fieldexcludes=Forbidden"` |

### 条件必填验证器

| 标签 | 描述 | 示例 |
|------|------|------|
| `required_if=Field value` | 如果字段等于值则必填 | `validate:"required_if=Status active"` |
| `required_unless=Field value` | 除非字段等于值否则必填 | `validate:"required_unless=Type guest"` |
| `required_with=Field` | 如果字段存在则必填 | `validate:"required_with=Email"` |
| `required_without=Field` | 如果字段不存在则必填 | `validate:"required_without=Phone"` |

**示例：**

```go
type User struct {
    Role    string
    // 当 Role 为 "admin" 时 Profile 为必填，其他角色为可选
    Profile string `validate:"required_if=Role admin"`
    // 除非 Role 为 "guest"，否则 Bio 为必填
    Bio     string `validate:"required_unless=Role guest"`
}

type Contact struct {
    Email string
    Phone string
    // 当 Email 非空时 Name 为必填
    Name  string `validate:"required_with=Email"`
    // Email 和 BackupEmail 至少需要提供一个
    BackupEmail string `validate:"required_without=Email"`
}
```

## 支持的文件格式

| 格式 | 扩展名 | 压缩格式 |
|------|--------|----------|
| CSV | `.csv` | `.csv.gz`、`.csv.bz2`、`.csv.xz`、`.csv.zst`、`.csv.z`、`.csv.snappy`、`.csv.s2`、`.csv.lz4` |
| TSV | `.tsv` | `.tsv.gz`、`.tsv.bz2`、`.tsv.xz`、`.tsv.zst`、`.tsv.z`、`.tsv.snappy`、`.tsv.s2`、`.tsv.lz4` |
| LTSV | `.ltsv` | `.ltsv.gz`、`.ltsv.bz2`、`.ltsv.xz`、`.ltsv.zst`、`.ltsv.z`、`.ltsv.snappy`、`.ltsv.s2`、`.ltsv.lz4` |
| JSON | `.json` | `.json.gz`、`.json.bz2`、`.json.xz`、`.json.zst`、`.json.z`、`.json.snappy`、`.json.s2`、`.json.lz4` |
| JSONL | `.jsonl` | `.jsonl.gz`、`.jsonl.bz2`、`.jsonl.xz`、`.jsonl.zst`、`.jsonl.z`、`.jsonl.snappy`、`.jsonl.s2`、`.jsonl.lz4` |
| Excel | `.xlsx` | `.xlsx.gz`、`.xlsx.bz2`、`.xlsx.xz`、`.xlsx.zst`、`.xlsx.z`、`.xlsx.snappy`、`.xlsx.s2`、`.xlsx.lz4` |
| Parquet | `.parquet` | `.parquet.gz`、`.parquet.bz2`、`.parquet.xz`、`.parquet.zst`、`.parquet.z`、`.parquet.snappy`、`.parquet.s2`、`.parquet.lz4` |

### 支持的压缩格式

| 格式 | 扩展名 | 描述 |
|------|--------|------|
| gzip | `.gz` | GNU zip 压缩，广泛使用 |
| bzip2 | `.bz2` | 块排序压缩，压缩率优秀 |
| xz | `.xz` | LZMA2 高性能压缩 |
| zstd | `.zst` | Facebook 的 Zstandard 压缩 |
| zlib | `.z` | 标准 DEFLATE 压缩 |
| snappy | `.snappy` | Google 的高速压缩 |
| s2 | `.s2` | 改进的 Snappy 扩展 |
| lz4 | `.lz4` | 极速压缩 |

**Parquet 压缩说明**：外部压缩（`.parquet.gz` 等）是针对容器文件本身的。Parquet 文件还可能使用内部压缩（Snappy、GZIP、LZ4、ZSTD），这由 parquet-go 库透明处理。

## 与 filesql 集成

```go
// 使用预处理和验证处理文件
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// 检查验证错误
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("第 %d 行，列 %s：%s", e.Row, e.Column, e.Message)
    }
}

// 使用 Builder 模式将预处理的数据传递给 filesql
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

// 对预处理的数据执行 SQL 查询
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## 处理器选项

`NewProcessor` 接受函数选项来自定义行为：

### WithStrictTagParsing

默认情况下，无效的标签参数（例如，在期望数字的地方使用 `eq=abc`）会被静默忽略。启用严格模式可以检测这些配置错误：

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithStrictTagParsing())
var records []MyRecord

// 如果标签参数无效（例如："eq=abc"、"truncate=xyz"），将返回错误
_, _, err := processor.Process(input, &records)
```

### WithValidRowsOnly

默认情况下，输出包含所有行（有效和无效）。使用 `WithValidRowsOnly` 将输出过滤为仅包含有效行：

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV, fileprep.WithValidRowsOnly())
var records []MyRecord

reader, result, err := processor.Process(input, &records)
// reader 仅包含通过所有验证的行
// records 仅包含有效的结构体
// result.RowCount 包含所有行；result.ValidRowCount 包含有效行数
// result.Errors 仍然报告所有验证失败
```

选项可以组合使用：

```go
processor := fileprep.NewProcessor(fileprep.FileTypeCSV,
    fileprep.WithStrictTagParsing(),
    fileprep.WithValidRowsOnly(),
)
```

## 设计考虑

### 基于名称的列绑定

结构体字段**按名称**而非按位置映射到文件列。字段名自动转换为 `snake_case` 以匹配列标题。文件中的列顺序无关紧要。

```go
type User struct {
    UserName string `name:"user"`       // 匹配 "user" 列（而非 "user_name"）
    Email    string `name:"mail_addr"`  // 匹配 "mail_addr" 列（而非 "email"）
    Age      string                     // 匹配 "age" 列（自动 snake_case）
}
```

如果您的 LTSV 键使用连字符（`user-id`）或 Parquet/XLSX 列使用驼峰命名（`userId`），请使用 `name` 标签指定确切的列名。

有关大小写敏感规则、重复标题行为和缺失列处理，请参阅 [使用 fileprep 之前](#使用-fileprep-之前)。

### 内存使用

fileprep 将**整个文件加载到内存**中进行处理。这允许随机访问和多遍操作，但对大文件有影响：

| 文件大小 | 预计内存 | 建议 |
|----------|----------|------|
| < 100 MB | 约为文件大小的 2-3 倍 | 直接处理 |
| 100-500 MB | 500 MB - 1.5 GB | 监控内存，考虑分块处理 |
| > 500 MB | > 1.5 GB | 拆分文件或使用流式替代方案 |

对于压缩输入（gzip、bzip2、xz、zstd、zlib、snappy、s2、lz4），内存使用基于**解压后**的大小。

## 性能

处理包含 21 列复杂结构的 CSV 文件的基准测试结果。每个字段使用多个预处理和验证标签：

**使用的预处理标签：** trim、lowercase、uppercase、keep_digits、pad_left、strip_html、strip_newline、collapse_space、truncate、fix_scheme、default

**使用的验证标签：** required、alpha、numeric、email、uuid、ip_addr、url、oneof、min、max、len、printascii、ascii、eqfield

| 记录数 | 时间 | 内存 | 分配次数 |
|-------:|-----:|-----:|---------:|
| 100 | 0.6 ms | 0.9 MB | 7,654 |
| 1,000 | 6.1 ms | 9.6 MB | 74,829 |
| 10,000 | 69 ms | 101 MB | 746,266 |
| 50,000 | 344 ms | 498 MB | 3,690,281 |

```bash
# 快速基准测试（100 和 1,000 条记录）
make bench

# 完整基准测试（所有大小包括 50,000 条记录）
make bench-all
```

*在 AMD Ryzen AI MAX+ 395、Go 1.24、Linux 上测试。结果因硬件而异。*

## 相关或启发项目

- [nao1215/filesql](https://github.com/nao1215/filesql) - CSV、TSV、LTSV、Parquet、Excel 的 SQL 驱动，支持 gzip、bzip2、xz、zstd。
- [nao1215/fileframe](https://github.com/nao1215/fileframe) - CSV/TSV/LTSV、Parquet、Excel 的 DataFrame API。
- [nao1215/csv](https://github.com/nao1215/csv) - 带验证和简单 DataFrame 的 Go CSV 读取库。
- [go-playground/validator](https://github.com/go-playground/validator) - Go 结构体和字段验证，包括跨字段、跨结构体、Map、Slice 和 Array 验证
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - go-header-csv 是带标题的 CSV 编码器/解码器。

## 贡献

欢迎贡献！详情请参阅 [Contributing Guide](../../CONTRIBUTING.md)。

## 支持

如果您觉得这个项目有用，请考虑：

- 在 GitHub 上给个星 - 这有助于其他人发现这个项目
- [成为赞助者](https://github.com/sponsors/nao1215) - 您的支持帮助维护项目并激励持续开发

您的支持，无论是星标、赞助还是代码贡献，都是推动这个项目前进的动力。谢谢！

## 许可证

本项目采用 MIT 许可证 - 详情请参阅 [LICENSE](../../LICENSE) 文件。
