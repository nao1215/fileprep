# fileprep

[![Go Reference](https://pkg.go.dev/badge/github.com/nao1215/fileprep.svg)](https://pkg.go.dev/github.com/nao1215/fileprep)
[![Go Report Card](https://goreportcard.com/badge/github.com/nao1215/fileprep)](https://goreportcard.com/report/github.com/nao1215/fileprep)
[![MultiPlatformUnitTest](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml/badge.svg)](https://github.com/nao1215/fileprep/actions/workflows/unit_test.yml)
![Coverage](https://raw.githubusercontent.com/nao1215/octocovs-central-repo/main/badges/nao1215/fileprep/coverage.svg)

[English](../../README.md) | [日本語](../ja/README.md) | [Español](../es/README.md) | [한국어](../ko/README.md) | [Русский](../ru/README.md) | [中文](../zh-cn/README.md)

![fileprep-logo](../images/fileprep-logo-small.png)

**fileprep** est une bibliothèque Go pour nettoyer, normaliser et valider des données structurées (CSV, TSV, LTSV, JSON, JSONL, Parquet et Excel) via des règles légères basées sur les balises struct, avec un support transparent des flux gzip, bzip2, xz, zstd, zlib, snappy, s2 et lz4.

## Pourquoi fileprep ?

J'ai développé [nao1215/filesql](https://github.com/nao1215/filesql), qui permet d'exécuter des requêtes SQL sur des fichiers comme CSV, TSV, LTSV, Parquet et Excel. J'ai également créé [nao1215/csv](https://github.com/nao1215/csv) pour la validation des fichiers CSV.

En étudiant l'apprentissage automatique, j'ai réalisé : "Si j'étends [nao1215/csv](https://github.com/nao1215/csv) pour supporter les mêmes formats de fichiers que [nao1215/filesql](https://github.com/nao1215/filesql), je pourrais les combiner pour effectuer des opérations de type ETL". Cette idée a conduit à la création de **fileprep** : une bibliothèque qui fait le pont entre le prétraitement/validation des données et les requêtes SQL sur fichiers.

## Fonctionnalités

- Support multi-formats : CSV, TSV, LTSV, JSON (.json), JSONL (.jsonl), Parquet, Excel (.xlsx)
- Support de la compression : gzip (.gz), bzip2 (.bz2), xz (.xz), zstd (.zst), zlib (.z), snappy (.snappy), s2 (.s2), lz4 (.lz4)
- Liaison de colonnes par nom : Les champs correspondent automatiquement aux noms de colonnes en `snake_case`, personnalisable via la balise `name`
- Prétraitement basé sur les balises struct (`prep`) : trim, lowercase, uppercase, valeurs par défaut
- Validation basée sur les balises struct (`validate`) : required et plus
- Intégration [filesql](https://github.com/nao1215/filesql) : Retourne `io.Reader` pour une utilisation directe avec filesql
- Rapport d'erreurs détaillé : Informations de ligne et colonne pour chaque erreur

## Installation

```bash
go get github.com/nao1215/fileprep
```

## Prérequis

- Version Go : 1.24 ou ultérieure
- Systèmes d'exploitation :
  - Linux
  - macOS
  - Windows


## Démarrage Rapide

```go
package main

import (
    "fmt"
    "os"
    "strings"

    "github.com/nao1215/fileprep"
)

// User représente un enregistrement utilisateur avec prétraitement et validation
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
        fmt.Printf("Erreur : %v\n", err)
        return
    }

    fmt.Printf("Traité %d lignes, %d valides\n", result.RowCount, result.ValidRowCount)

    for _, user := range users {
        fmt.Printf("Nom : %q, Email : %q\n", user.Name, user.Email)
    }

    // reader peut être passé directement à filesql
    _ = reader
}
```

Sortie :
```
Traité 2 lignes, 2 valides
Nom : "John Doe", Email : "john@example.com"
Nom : "Jane Smith", Email : "jane@example.com"
```

## Exemples Avancés

### Prétraitement et Validation de Données Complexes

Cet exemple démontre toute la puissance de fileprep : en combinant plusieurs préprocesseurs et validateurs pour nettoyer et valider des données réelles désordonnées.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Employee représente les données d'employé avec prétraitement et validation complets
type Employee struct {
    // ID : compléter à 6 chiffres, doit être numérique
    EmployeeID string `name:"id" prep:"trim,pad_left=6:0" validate:"required,numeric,len=6"`

    // Nom : nettoyer les espaces, requis alphabétique avec espaces
    FullName string `name:"name" prep:"trim,collapse_space" validate:"required,alphaspace"`

    // Email : normaliser en minuscules, valider le format
    Email string `prep:"trim,lowercase" validate:"required,email"`

    // Département : normaliser en majuscules, doit être une des valeurs autorisées
    Department string `prep:"trim,uppercase" validate:"required,oneof=ENGINEERING SALES MARKETING HR"`

    // Salaire : garder uniquement les chiffres, valider la plage
    Salary string `prep:"trim,keep_digits" validate:"required,numeric,gte=30000,lte=500000"`

    // Téléphone : extraire les chiffres, valider le format E.164 après ajout du code pays
    Phone string `prep:"trim,keep_digits,prefix=+1" validate:"e164"`

    // Date de début : valider le format datetime
    StartDate string `name:"start_date" prep:"trim" validate:"required,datetime=2006-01-02"`

    // ID Manager : requis seulement si le département n'est pas HR
    ManagerID string `name:"manager_id" prep:"trim,pad_left=6:0" validate:"required_unless=Department HR"`

    // Site web : corriger le schéma manquant, valider l'URL
    Website string `prep:"trim,lowercase,fix_scheme=https" validate:"url"`
}

func main() {
    // Données CSV désordonnées du monde réel
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
        fmt.Printf("Erreur fatale : %v\n", err)
        return
    }

    fmt.Printf("=== Résultat du Traitement ===\n")
    fmt.Printf("Lignes totales : %d, Lignes valides : %d\n\n", result.RowCount, result.ValidRowCount)

    for i, emp := range employees {
        fmt.Printf("Employé %d :\n", i+1)
        fmt.Printf("  ID :          %s\n", emp.EmployeeID)
        fmt.Printf("  Nom :         %s\n", emp.FullName)
        fmt.Printf("  Email :       %s\n", emp.Email)
        fmt.Printf("  Département : %s\n", emp.Department)
        fmt.Printf("  Salaire :     %s\n", emp.Salary)
        fmt.Printf("  Téléphone :   %s\n", emp.Phone)
        fmt.Printf("  Date Début :  %s\n", emp.StartDate)
        fmt.Printf("  ID Manager :  %s\n", emp.ManagerID)
        fmt.Printf("  Site Web :    %s\n\n", emp.Website)
    }
}
```

Sortie :
```
=== Résultat du Traitement ===
Lignes totales : 4, Lignes valides : 4

Employé 1 :
  ID :          000042
  Nom :         John Doe
  Email :       john.doe@company.com
  Département : ENGINEERING
  Salaire :     75000
  Téléphone :   +15551234567
  Date Début :  2023-01-15
  ID Manager :  000001
  Site Web :    https://company.com/john

Employé 2 :
  ID :          000007
  Nom :         Jane Smith
  Email :       jane@company.com
  Département : SALES
  Salaire :     120000
  Téléphone :   +15559876543
  Date Début :  2022-06-01
  ID Manager :  000002
  Site Web :    https://www.linkedin.com/in/jane

Employé 3 :
  ID :          000123
  Nom :         Bob Wilson
  Email :       bob.wilson@company.com
  Département : HR
  Salaire :     45000
  Téléphone :   +15551112222
  Date Début :  2024-03-20
  ID Manager :  000000
  Site Web :

Employé 4 :
  ID :          000099
  Nom :         Alice Brown
  Email :       alice@company.com
  Département : MARKETING
  Salaire :     88500
  Téléphone :   +15554443333
  Date Début :  2023-09-10
  ID Manager :  000003
  Site Web :    https://alice.dev
```


### Rapport d'Erreurs Détaillé

Lorsque la validation échoue, fileprep fournit des informations d'erreur précises incluant le numéro de ligne, le nom de colonne et la raison spécifique de l'échec de validation.

```go
package main

import (
    "fmt"
    "strings"

    "github.com/nao1215/fileprep"
)

// Order représente une commande avec des règles de validation strictes
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
    // CSV avec plusieurs erreurs de validation
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
        fmt.Printf("Erreur fatale : %v\n", err)
        return
    }

    fmt.Printf("=== Rapport de Validation ===\n")
    fmt.Printf("Lignes totales :   %d\n", result.RowCount)
    fmt.Printf("Lignes valides :   %d\n", result.ValidRowCount)
    fmt.Printf("Lignes invalides : %d\n", result.RowCount-result.ValidRowCount)
    fmt.Printf("Erreurs totales :  %d\n\n", len(result.ValidationErrors()))

    if result.HasErrors() {
        fmt.Println("=== Détails des Erreurs ===")
        for _, e := range result.ValidationErrors() {
            fmt.Printf("Ligne %d, Colonne '%s' : %s\n", e.Row, e.Column, e.Message)
        }
    }
}
```

Sortie :
```
=== Rapport de Validation ===
Lignes totales :   4
Lignes valides :   1
Lignes invalides : 3
Erreurs totales :  23

=== Détails des Erreurs ===
Ligne 2, Colonne 'order_id' : value must be a valid UUID version 4
Ligne 2, Colonne 'customer_id' : value must be numeric
Ligne 2, Colonne 'email' : value must be a valid email address
Ligne 2, Colonne 'amount' : value must be greater than 0
Ligne 2, Colonne 'currency' : value must have exactly 3 characters
Ligne 2, Colonne 'country' : value must have exactly 2 characters
Ligne 2, Colonne 'order_date' : value must be a valid datetime in format: 2006-01-02
Ligne 2, Colonne 'ip_address' : value must be a valid IP address
Ligne 2, Colonne 'promo_code' : value must contain only alphanumeric characters
Ligne 2, Colonne 'quantity' : value must be greater than or equal to 1
Ligne 2, Colonne 'unit_price' : value must be greater than 0
Ligne 2, Colonne 'ship_date' : value must be greater than field OrderDate
Ligne 2, Colonne 'total_check' : value must equal field Amount
Ligne 3, Colonne 'customer_id' : value is required
Ligne 3, Colonne 'email' : value must be a valid email address
Ligne 3, Colonne 'amount' : value must be less than or equal to 10000
Ligne 3, Colonne 'currency' : value must have exactly 3 characters
Ligne 3, Colonne 'country' : value must contain only alphabetic characters
Ligne 3, Colonne 'order_date' : value must be a valid datetime in format: 2006-01-02
Ligne 3, Colonne 'quantity' : value must be less than or equal to 100
Ligne 3, Colonne 'unit_price' : value must be greater than 0
Ligne 3, Colonne 'ship_date' : value must be greater than field OrderDate
Ligne 4, Colonne 'ship_date' : value must be greater than field OrderDate
```

## Balises de Prétraitement (`prep`)

Plusieurs balises peuvent être combinées : `prep:"trim,lowercase,default=N/A"`

### Préprocesseurs de Base

| Balise | Description | Exemple |
|--------|-------------|---------|
| `trim` | Supprimer les espaces au début/fin | `prep:"trim"` |
| `ltrim` | Supprimer les espaces au début | `prep:"ltrim"` |
| `rtrim` | Supprimer les espaces à la fin | `prep:"rtrim"` |
| `lowercase` | Convertir en minuscules | `prep:"lowercase"` |
| `uppercase` | Convertir en majuscules | `prep:"uppercase"` |
| `default=value` | Définir une valeur par défaut si vide | `prep:"default=N/A"` |

### Transformation de Chaînes

| Balise | Description | Exemple |
|--------|-------------|---------|
| `replace=old:new` | Remplacer toutes les occurrences | `prep:"replace=;:,"` |
| `prefix=value` | Ajouter une chaîne au début | `prep:"prefix=ID_"` |
| `suffix=value` | Ajouter une chaîne à la fin | `prep:"suffix=_END"` |
| `truncate=N` | Limiter à N caractères | `prep:"truncate=100"` |
| `strip_html` | Supprimer les balises HTML | `prep:"strip_html"` |
| `strip_newline` | Supprimer les sauts de ligne (LF, CRLF, CR) | `prep:"strip_newline"` |
| `collapse_space` | Réduire les espaces multiples en un seul | `prep:"collapse_space"` |

### Filtrage de Caractères

| Balise | Description | Exemple |
|--------|-------------|---------|
| `remove_digits` | Supprimer tous les chiffres | `prep:"remove_digits"` |
| `remove_alpha` | Supprimer tous les caractères alphabétiques | `prep:"remove_alpha"` |
| `keep_digits` | Garder uniquement les chiffres | `prep:"keep_digits"` |
| `keep_alpha` | Garder uniquement les caractères alphabétiques | `prep:"keep_alpha"` |
| `trim_set=chars` | Supprimer les caractères spécifiés des deux extrémités | `prep:"trim_set=@#$"` |

### Remplissage

| Balise | Description | Exemple |
|--------|-------------|---------|
| `pad_left=N:char` | Remplir à gauche jusqu'à N caractères | `prep:"pad_left=5:0"` |
| `pad_right=N:char` | Remplir à droite jusqu'à N caractères | `prep:"pad_right=10: "` |

### Préprocesseurs Avancés

| Balise | Description | Exemple |
|--------|-------------|---------|
| `normalize_unicode` | Normaliser Unicode au format NFC | `prep:"normalize_unicode"` |
| `nullify=value` | Traiter une chaîne spécifique comme vide | `prep:"nullify=NULL"` |
| `coerce=type` | Coercition de type (int, float, bool) | `prep:"coerce=int"` |
| `fix_scheme=scheme` | Ajouter ou corriger le schéma d'URL | `prep:"fix_scheme=https"` |
| `regex_replace=pattern:replacement` | Remplacement basé sur regex | `prep:"regex_replace=\\d+:X"` |

## Balises de Validation (`validate`)

Plusieurs balises peuvent être combinées : `validate:"required,email"`

### Validateurs de Base

| Balise | Description | Exemple |
|--------|-------------|---------|
| `required` | Le champ ne doit pas être vide | `validate:"required"` |
| `boolean` | Doit être true, false, 0 ou 1 | `validate:"boolean"` |

### Validateurs de Type de Caractère

| Balise | Description | Exemple |
|--------|-------------|---------|
| `alpha` | Caractères alphabétiques ASCII uniquement | `validate:"alpha"` |
| `alphaunicode` | Lettres Unicode uniquement | `validate:"alphaunicode"` |
| `alphaspace` | Caractères alphabétiques ou espaces | `validate:"alphaspace"` |
| `alphanumeric` | Caractères alphanumériques ASCII | `validate:"alphanumeric"` |
| `alphanumunicode` | Lettres ou chiffres Unicode | `validate:"alphanumunicode"` |
| `numeric` | Entier valide | `validate:"numeric"` |
| `number` | Nombre valide (entier ou décimal) | `validate:"number"` |
| `ascii` | Caractères ASCII uniquement | `validate:"ascii"` |
| `printascii` | Caractères ASCII imprimables (0x20-0x7E) | `validate:"printascii"` |
| `multibyte` | Contient des caractères multi-octets | `validate:"multibyte"` |

### Validateurs de Comparaison Numérique

| Balise | Description | Exemple |
|--------|-------------|---------|
| `eq=N` | La valeur est égale à N | `validate:"eq=100"` |
| `ne=N` | La valeur n'est pas égale à N | `validate:"ne=0"` |
| `gt=N` | La valeur est supérieure à N | `validate:"gt=0"` |
| `gte=N` | La valeur est supérieure ou égale à N | `validate:"gte=1"` |
| `lt=N` | La valeur est inférieure à N | `validate:"lt=100"` |
| `lte=N` | La valeur est inférieure ou égale à N | `validate:"lte=99"` |
| `min=N` | La valeur est au moins N | `validate:"min=0"` |
| `max=N` | La valeur est au maximum N | `validate:"max=100"` |
| `len=N` | Exactement N caractères | `validate:"len=10"` |

### Validateurs de Chaînes

| Balise | Description | Exemple |
|--------|-------------|---------|
| `oneof=a b c` | La valeur est l'une des valeurs autorisées | `validate:"oneof=active inactive"` |
| `lowercase` | La valeur est entièrement en minuscules | `validate:"lowercase"` |
| `uppercase` | La valeur est entièrement en majuscules | `validate:"uppercase"` |
| `eq_ignore_case=value` | Égalité insensible à la casse | `validate:"eq_ignore_case=yes"` |
| `ne_ignore_case=value` | Inégalité insensible à la casse | `validate:"ne_ignore_case=no"` |

### Validateurs de Contenu de Chaînes

| Balise | Description | Exemple |
|--------|-------------|---------|
| `startswith=prefix` | La valeur commence par le préfixe | `validate:"startswith=http"` |
| `startsnotwith=prefix` | La valeur ne commence pas par le préfixe | `validate:"startsnotwith=_"` |
| `endswith=suffix` | La valeur se termine par le suffixe | `validate:"endswith=.com"` |
| `endsnotwith=suffix` | La valeur ne se termine pas par le suffixe | `validate:"endsnotwith=.tmp"` |
| `contains=substr` | La valeur contient la sous-chaîne | `validate:"contains=@"` |
| `containsany=chars` | La valeur contient l'un des caractères | `validate:"containsany=abc"` |
| `containsrune=r` | La valeur contient le rune | `validate:"containsrune=@"` |
| `excludes=substr` | La valeur ne contient pas la sous-chaîne | `validate:"excludes=admin"` |
| `excludesall=chars` | La valeur ne contient aucun des caractères | `validate:"excludesall=<>"` |
| `excludesrune=r` | La valeur ne contient pas le rune | `validate:"excludesrune=$"` |

### Validateurs de Format

| Balise | Description | Exemple |
|--------|-------------|---------|
| `email` | Adresse email valide | `validate:"email"` |
| `uri` | URI valide | `validate:"uri"` |
| `url` | URL valide | `validate:"url"` |
| `http_url` | URL HTTP ou HTTPS valide | `validate:"http_url"` |
| `https_url` | URL HTTPS valide | `validate:"https_url"` |
| `url_encoded` | Chaîne encodée URL | `validate:"url_encoded"` |
| `datauri` | Data URI valide | `validate:"datauri"` |
| `datetime=layout` | Datetime valide correspondant au layout Go | `validate:"datetime=2006-01-02"` |
| `uuid` | UUID valide (toute version) | `validate:"uuid"` |
| `uuid3` | UUID version 3 valide | `validate:"uuid3"` |
| `uuid4` | UUID version 4 valide | `validate:"uuid4"` |
| `uuid5` | UUID version 5 valide | `validate:"uuid5"` |
| `ulid` | ULID valide | `validate:"ulid"` |
| `e164` | Numéro de téléphone E.164 valide | `validate:"e164"` |
| `latitude` | Latitude valide (-90 à 90) | `validate:"latitude"` |
| `longitude` | Longitude valide (-180 à 180) | `validate:"longitude"` |
| `hexadecimal` | Chaîne hexadécimale valide | `validate:"hexadecimal"` |
| `hexcolor` | Code couleur hexadécimal valide | `validate:"hexcolor"` |
| `rgb` | Couleur RGB valide | `validate:"rgb"` |
| `rgba` | Couleur RGBA valide | `validate:"rgba"` |
| `hsl` | Couleur HSL valide | `validate:"hsl"` |
| `hsla` | Couleur HSLA valide | `validate:"hsla"` |

### Validateurs Réseau

| Balise | Description | Exemple |
|--------|-------------|---------|
| `ip_addr` | Adresse IP valide (v4 ou v6) | `validate:"ip_addr"` |
| `ip4_addr` | Adresse IPv4 valide | `validate:"ip4_addr"` |
| `ip6_addr` | Adresse IPv6 valide | `validate:"ip6_addr"` |
| `cidr` | Notation CIDR valide | `validate:"cidr"` |
| `cidrv4` | CIDR IPv4 valide | `validate:"cidrv4"` |
| `cidrv6` | CIDR IPv6 valide | `validate:"cidrv6"` |
| `mac` | Adresse MAC valide | `validate:"mac"` |
| `fqdn` | Nom de domaine pleinement qualifié valide | `validate:"fqdn"` |
| `hostname` | Nom d'hôte valide (RFC 952) | `validate:"hostname"` |
| `hostname_rfc1123` | Nom d'hôte valide (RFC 1123) | `validate:"hostname_rfc1123"` |
| `hostname_port` | hostname:port valide | `validate:"hostname_port"` |

### Validateurs Inter-champs

| Balise | Description | Exemple |
|--------|-------------|---------|
| `eqfield=Field` | La valeur est égale à un autre champ | `validate:"eqfield=Password"` |
| `nefield=Field` | La valeur n'est pas égale à un autre champ | `validate:"nefield=OldPassword"` |
| `gtfield=Field` | La valeur est supérieure à un autre champ | `validate:"gtfield=MinPrice"` |
| `gtefield=Field` | La valeur est >= un autre champ | `validate:"gtefield=StartDate"` |
| `ltfield=Field` | La valeur est inférieure à un autre champ | `validate:"ltfield=MaxPrice"` |
| `ltefield=Field` | La valeur est <= un autre champ | `validate:"ltefield=EndDate"` |
| `fieldcontains=Field` | La valeur contient la valeur d'un autre champ | `validate:"fieldcontains=Keyword"` |
| `fieldexcludes=Field` | La valeur exclut la valeur d'un autre champ | `validate:"fieldexcludes=Forbidden"` |

### Validateurs de Requis Conditionnel

| Balise | Description | Exemple |
|--------|-------------|---------|
| `required_if=Field value` | Requis si le champ est égal à value | `validate:"required_if=Status active"` |
| `required_unless=Field value` | Requis sauf si le champ est égal à value | `validate:"required_unless=Type guest"` |
| `required_with=Field` | Requis si le champ est présent | `validate:"required_with=Email"` |
| `required_without=Field` | Requis si le champ est absent | `validate:"required_without=Phone"` |

## Formats de Fichiers Supportés

| Format | Extension | Compressé |
|--------|-----------|-----------|
| CSV | `.csv` | `.csv.gz`, `.csv.bz2`, `.csv.xz`, `.csv.zst`, `.csv.z`, `.csv.snappy`, `.csv.s2`, `.csv.lz4` |
| TSV | `.tsv` | `.tsv.gz`, `.tsv.bz2`, `.tsv.xz`, `.tsv.zst`, `.tsv.z`, `.tsv.snappy`, `.tsv.s2`, `.tsv.lz4` |
| LTSV | `.ltsv` | `.ltsv.gz`, `.ltsv.bz2`, `.ltsv.xz`, `.ltsv.zst`, `.ltsv.z`, `.ltsv.snappy`, `.ltsv.s2`, `.ltsv.lz4` |
| JSON | `.json` | `.json.gz`, `.json.bz2`, `.json.xz`, `.json.zst`, `.json.z`, `.json.snappy`, `.json.s2`, `.json.lz4` |
| JSONL | `.jsonl` | `.jsonl.gz`, `.jsonl.bz2`, `.jsonl.xz`, `.jsonl.zst`, `.jsonl.z`, `.jsonl.snappy`, `.jsonl.s2`, `.jsonl.lz4` |
| Excel | `.xlsx` | `.xlsx.gz`, `.xlsx.bz2`, `.xlsx.xz`, `.xlsx.zst`, `.xlsx.z`, `.xlsx.snappy`, `.xlsx.s2`, `.xlsx.lz4` |
| Parquet | `.parquet` | `.parquet.gz`, `.parquet.bz2`, `.parquet.xz`, `.parquet.zst`, `.parquet.z`, `.parquet.snappy`, `.parquet.s2`, `.parquet.lz4` |

### Formats de Compression Supportés

| Format | Extension | Description |
|--------|-----------|-------------|
| gzip | `.gz` | Compression GNU zip, largement utilisée |
| bzip2 | `.bz2` | Compression par bloc avec excellent taux |
| xz | `.xz` | Compression LZMA2 haute performance |
| zstd | `.zst` | Compression Zstandard de Facebook |
| zlib | `.z` | Compression DEFLATE standard |
| snappy | `.snappy` | Compression rapide de Google |
| s2 | `.s2` | Extension Snappy améliorée |
| lz4 | `.lz4` | Compression extrêmement rapide |

**Note sur la compression Parquet** : La compression externe (`.parquet.gz`, etc.) est pour le fichier conteneur lui-même. Les fichiers Parquet peuvent également utiliser une compression interne (Snappy, GZIP, LZ4, ZSTD) qui est gérée de manière transparente par la bibliothèque parquet-go.

**Note sur les fichiers Excel** : Seule la **première feuille** est traitée. Les feuilles suivantes dans les classeurs multi-feuilles seront ignorées.

**Note sur les fichiers JSON/JSONL** : Les données JSON/JSONL sont stockées dans une seule colonne `"data"` contenant des chaînes JSON brutes. Chaque élément d'un tableau JSON ou ligne JSONL devient une ligne. L'entrée JSON est sortie en JSONL compact (une valeur JSON par ligne). Les balises de prétraitement opèrent sur la chaîne JSON brute, pas sur les champs individuels qu'elle contient. Si le prétraitement détruit la structure JSON, `Process` retourne `ErrInvalidJSONAfterPrep`. Si toutes les lignes deviennent vides après prétraitement, `Process` retourne `ErrEmptyJSONOutput`.

## Intégration avec filesql

```go
// Traiter le fichier avec prétraitement et validation
processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
var records []MyRecord

reader, result, err := processor.Process(file, &records)
if err != nil {
    return err
}

// Vérifier les erreurs de validation
if result.HasErrors() {
    for _, e := range result.ValidationErrors() {
        log.Printf("Ligne %d, Colonne %s : %s", e.Row, e.Column, e.Message)
    }
}

// Passer les données prétraitées à filesql en utilisant le pattern Builder
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

// Exécuter des requêtes SQL sur les données prétraitées
rows, err := db.QueryContext(ctx, "SELECT * FROM my_table WHERE age > 20")
```

## Considérations de Conception

### Liaison de Colonnes par Nom

Les champs struct sont mappés aux colonnes du fichier **par nom**, pas par position. Les noms de champs sont automatiquement convertis en `snake_case` pour correspondre aux en-têtes de colonne CSV :

```go
// Colonnes du fichier : user_name, email_address, phone_number (n'importe quel ordre)
type User struct {
    UserName     string  // → correspond à la colonne "user_name"
    EmailAddress string  // → correspond à la colonne "email_address"
    PhoneNumber  string  // → correspond à la colonne "phone_number"
}
```

**L'ordre des colonnes n'a pas d'importance** - les champs sont associés par nom, vous pouvez donc réorganiser les colonnes dans votre CSV sans changer votre struct.

#### Noms de Colonnes Personnalisés avec la Balise `name`

Utilisez la balise `name` pour remplacer le nom de colonne généré automatiquement :

```go
type User struct {
    UserName string `name:"user"`       // → correspond à la colonne "user" (pas "user_name")
    Email    string `name:"mail_addr"`  // → correspond à la colonne "mail_addr" (pas "email")
    Age      string                     // → correspond à la colonne "age" (snake_case automatique)
}
```

#### Comportement des Colonnes Manquantes

Si une colonne CSV n'existe pas pour un champ struct, la valeur du champ est traitée comme une chaîne vide. La validation s'exécute toujours, donc `required` détectera les colonnes manquantes :

```go
type User struct {
    Name    string `validate:"required"`  // Erreur si la colonne "name" est manquante
    Country string                        // Chaîne vide si la colonne "country" est manquante
}
```

#### Sensibilité à la Casse et En-têtes en Double

**La correspondance des en-têtes est sensible à la casse et exacte.** Un champ struct `UserName` se mappe à `user_name`, mais des en-têtes comme `User_Name`, `USER_NAME` ou `userName` **ne** correspondront **pas** :

```go
type User struct {
    UserName string  // ✓ correspond à "user_name"
                     // ✗ NE correspond PAS à "User_Name", "USER_NAME", "userName"
}
```

Cela s'applique à tous les formats de fichiers : CSV, TSV, clés LTSV et noms de colonnes Parquet/XLSX doivent correspondre exactement.

**Noms de colonnes en double :** Si un fichier contient des noms d'en-tête en double (par ex., `id,id,name`), la **première occurrence** est utilisée pour la liaison :

```csv
id,id,name
first,second,John  → struct.ID = "first" (la première colonne "id" gagne)
```

#### Notes Spécifiques aux Formats

**LTSV, Parquet et XLSX** suivent les mêmes règles de correspondance sensibles à la casse. Les clés/noms de colonnes doivent correspondre exactement :

```go
type Record struct {
    UserID string                 // attend la clé/colonne "user_id"
    Email  string `name:"EMAIL"`  // utilisez la balise name pour les colonnes non snake_case
}
```

Si vos clés LTSV utilisent des tirets (`user-id`) ou si les colonnes Parquet/XLSX utilisent camelCase (`userId`), utilisez la balise `name` pour spécifier le nom exact de la colonne.

### Utilisation Mémoire

fileprep charge le **fichier entier en mémoire** pour le traitement. Cela permet l'accès aléatoire et les opérations multi-passes, mais a des implications pour les gros fichiers :

| Taille du Fichier | Mémoire Approx. | Recommandation |
|-------------------|-----------------|----------------|
| < 100 Mo | ~2-3x taille du fichier | Traitement direct |
| 100-500 Mo | ~500 Mo - 1.5 Go | Surveiller la mémoire, considérer le fractionnement |
| > 500 Mo | > 1.5 Go | Diviser les fichiers ou utiliser des alternatives en streaming |

Pour les entrées compressées (gzip, bzip2, xz, zstd, zlib, snappy, s2, lz4), l'utilisation mémoire est basée sur la taille **décompressée**.

## Performance

Résultats de benchmark traitant des fichiers CSV avec un struct complexe contenant 21 colonnes. Chaque champ utilise plusieurs balises de prétraitement et validation :

**Balises de prétraitement utilisées :** trim, lowercase, uppercase, keep_digits, pad_left, strip_html, strip_newline, collapse_space, truncate, fix_scheme, default

**Balises de validation utilisées :** required, alpha, numeric, email, uuid, ip_addr, url, oneof, min, max, len, printascii, ascii, eqfield

| Enregistrements | Temps | Mémoire | Allocs/op |
|--------:|-----:|-------:|----------:|
| 100 | 0.6 ms | 0.9 Mo | 7 654 |
| 1 000 | 6.1 ms | 9.6 Mo | 74 829 |
| 10 000 | 69 ms | 101 Mo | 746 266 |
| 50 000 | 344 ms | 498 Mo | 3 690 281 |

```bash
# Benchmark rapide (100 et 1 000 enregistrements)
make bench

# Benchmark complet (toutes les tailles incluant 50 000 enregistrements)
make bench-all
```

*Testé sur AMD Ryzen AI MAX+ 395, Go 1.24, Linux. Les résultats varient selon le matériel.*

## Projets Liés ou d'Inspiration

- [nao1215/filesql](https://github.com/nao1215/filesql) - Driver SQL pour CSV, TSV, LTSV, Parquet, Excel avec support gzip, bzip2, xz, zstd.
- [nao1215/csv](https://github.com/nao1215/csv) - Lecture CSV avec validation et DataFrame simple en golang.
- [go-playground/validator](https://github.com/go-playground/validator) - Validation de Go Struct et Field, incluant Cross Field, Cross Struct, Map, Slice et Array diving.
- [shogo82148/go-header-csv](https://github.com/shogo82148/go-header-csv) - go-header-csv est un encodeur/décodeur csv avec en-tête.

## Contributions

Les contributions sont les bienvenues ! Veuillez consulter le [Guide de Contribution](../../CONTRIBUTING.md) pour plus de détails.

## Support

Si vous trouvez ce projet utile, veuillez considérer :

- Lui donner une étoile sur GitHub - cela aide les autres à découvrir le projet
- [Devenir sponsor](https://github.com/sponsors/nao1215) - votre soutien maintient le projet en vie et motive le développement continu

Votre soutien, que ce soit par des étoiles, des parrainages ou des contributions, est ce qui fait avancer ce projet. Merci !

## Licence

Ce projet est sous licence MIT - voir le fichier [LICENSE](../../LICENSE) pour plus de détails.
