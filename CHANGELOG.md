# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2025-12-11

### Added
- **New Compression Formats**: Added support for 4 new compression formats via fileparser v0.2.0
  - zlib (.z) - Standard DEFLATE compression
  - snappy (.snappy) - Google's high-speed compression
  - s2 (.s2) - Improved Snappy extension, faster
  - lz4 (.lz4) - Extremely fast compression
- **New FileType Constants**: Added 20 new FileType aliases for new compression format combinations
  - CSV: `FileTypeCSVZLIB`, `FileTypeCSVSNAPPY`, `FileTypeCSVS2`, `FileTypeCSVLZ4`
  - TSV: `FileTypeTSVZLIB`, `FileTypeTSVSNAPPY`, `FileTypeTSVS2`, `FileTypeTSVLZ4`
  - LTSV: `FileTypeLTSVZLIB`, `FileTypeLTSVSNAPPY`, `FileTypeLTSVS2`, `FileTypeLTSVLZ4`
  - Parquet: `FileTypeParquetZLIB`, `FileTypeParquetSNAPPY`, `FileTypeParquetS2`, `FileTypeParquetLZ4`
  - Excel: `FileTypeXLSXZLIB`, `FileTypeXLSXSNAPPY`, `FileTypeXLSXS2`, `FileTypeXLSXLZ4`
- **Integration Tests**: Added comprehensive tests for new compression formats (CSV, TSV, LTSV)

### Changed
- **Dependency Update**: Updated to fileparser v0.2.0 for new compression format support
- **Documentation**: Updated all README files (en, ja, es, fr, ko, ru, zh-cn) with new compression formats

## [0.3.0] - 2025-12-11

### Changed
- Migrated from `github.com/nao1215/filesql/parser` to `github.com/nao1215/fileparser` for file parsing
- Updated all internal references from `parser.` to `fileparser.`

### Removed
- Dependency on `github.com/nao1215/filesql`

## [0.2.0] - 2025-12-08

### Added
- **Conditional Required Validators** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): New validators for conditional field requirements
  - `required_if`: Required if another field equals a specific value
  - `required_unless`: Required unless another field equals a specific value
  - `required_with`: Required if another field is present
  - `required_without`: Required if another field is not present
- **Date/Time Validator** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): `datetime` validator with custom Go layout format support
- **Phone Number Validator** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): `e164` validator for E.164 international phone number format
- **Geolocation Validators** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): `latitude` (-90 to 90) and `longitude` (-180 to 180) validators
- **UUID Variant Validators** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): `uuid3`, `uuid4`, `uuid5` for specific UUID versions, and `ulid` for ULID format
- **Hexadecimal and Color Validators** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): `hexadecimal`, `hexcolor`, `rgb`, `rgba`, `hsl`, `hsla` validators
- **MAC Address Validator** ([9caa374](https://github.com/nao1215/fileprep/commit/9caa374)): `mac` validator for MAC address format
- **Advanced Examples** ([f771f9b](https://github.com/nao1215/fileprep/commit/f771f9b)): Comprehensive documentation examples
  - Complex Data Preprocessing and Validation example with real-world messy data
  - Detailed Error Reporting example demonstrating validation error handling
- **Benchmark Tests** ([607b868](https://github.com/nao1215/fileprep/commit/607b868)): Comprehensive benchmark suite for performance testing

### Changed
- **Performance Improvement** (PR [#6](https://github.com/nao1215/fileprep/pull/6), [607b868](https://github.com/nao1215/fileprep/commit/607b868)): ~10% faster processing through optimized preprocessing and validation pipeline
- **Documentation** ([f771f9b](https://github.com/nao1215/fileprep/commit/f771f9b)): Complete update of all README translations (Japanese, Spanish, French, Korean, Russian, Chinese) to match the English version with full feature documentation

## [0.1.0] - 2025-12-07

### Added
- **Initial Release**: First stable release of fileprep library
- **File Format Support**: CSV, TSV, LTSV, Parquet, Excel (.xlsx) with compression support (gzip, bzip2, xz, zstd)
- **Preprocessing Tags (`prep`)**: Comprehensive struct tag-based preprocessing
  - Basic preprocessors: `trim`, `ltrim`, `rtrim`, `lowercase`, `uppercase`, `default`
  - String transformation: `replace`, `prefix`, `suffix`, `truncate`, `strip_html`, `strip_newline`, `collapse_space`
  - Character filtering: `remove_digits`, `remove_alpha`, `keep_digits`, `keep_alpha`, `trim_set`
  - Padding: `pad_left`, `pad_right`
  - Advanced: `normalize_unicode`, `nullify`, `coerce`, `fix_scheme`, `regex_replace`
- **Validation Tags (`validate`)**: Compatible with go-playground/validator syntax
  - Basic validators: `required`, `boolean`
  - Character type validators: `alpha`, `alphaunicode`, `alphaspace`, `alphanumeric`, `alphanumunicode`, `numeric`, `number`, `ascii`, `printascii`, `multibyte`
  - Numeric comparison: `eq`, `ne`, `gt`, `gte`, `lt`, `lte`, `min`, `max`, `len`
  - String validators: `oneof`, `lowercase`, `uppercase`, `eq_ignore_case`, `ne_ignore_case`
  - String content: `startswith`, `startsnotwith`, `endswith`, `endsnotwith`, `contains`, `containsany`, `containsrune`, `excludes`, `excludesall`, `excludesrune`
  - Format validators: `email`, `uri`, `url`, `http_url`, `https_url`, `url_encoded`, `datauri`, `uuid`
  - Network validators: `ip_addr`, `ip4_addr`, `ip6_addr`, `cidr`, `cidrv4`, `cidrv6`, `fqdn`, `hostname`, `hostname_rfc1123`, `hostname_port`
  - Cross-field validators: `eqfield`, `nefield`, `gtfield`, `gtefield`, `ltfield`, `ltefield`, `fieldcontains`, `fieldexcludes`
- **Name-Based Column Binding**: Automatic snake_case conversion with `name` tag override
- **filesql Integration**: Returns `io.Reader` for direct use with filesql
- **Detailed Error Reporting**: Row and column information for each validation error

### Technical Details
- **Memory Optimization**: In-place record processing, pre-allocated output buffers, streaming parsers for CSV/TSV/LTSV
- **XLSX Streaming**: Uses excelize streaming API to reduce memory usage for large files
- **Parquet Buffer Reuse**: Reusable row buffer across row groups to reduce allocations
- **Format-Specific Limitations**:
  - XLSX: Only the first sheet is processed
  - LTSV: Maximum line size is 10MB
