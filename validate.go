package fileprep

import (
	"encoding/base64"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Regex patterns for validation
const (
	uuidRegexPattern    = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	dataURIRegexPattern = `^data:[^;]+;base64,[A-Za-z0-9+/]+={0,2}$`
	emailRegexPattern   = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	numberRegexPattern  = `^[-+]?[0-9]+(\.[0-9]+)?$`
	fileScheme          = "file"
)

// Common error messages (to avoid goconst warnings)
const (
	errMsgValidNumber       = "value must be a valid number"
	errMsgValidURL          = "value must be a valid URL"
	errMsgValidDataURI      = "value must be a valid data URI"
	errMsgValidFQDN         = "value must be a valid FQDN"
	errMsgValidHostnamePort = "value must be a valid hostname:port"
)

// Pre-compiled regexes
var (
	uuidRegex                 = regexp.MustCompile(uuidRegexPattern)
	dataURIRegex              = regexp.MustCompile(dataURIRegexPattern)
	emailRegex                = regexp.MustCompile(emailRegexPattern)
	numberRegex               = regexp.MustCompile(numberRegexPattern)
	urlEncodedRegex           = regexp.MustCompile(`^(?:[^%]|%[0-9A-Fa-f]{2})*$`)
	fqdnLabelRegex            = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	hostnameRFC952LabelRegex  = regexp.MustCompile(`^[A-Za-z](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
	hostnameRFC1123LabelRegex = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
)

// Validator defines the interface for validating values
type Validator interface {
	// Validate checks if the value is valid and returns an error message if not
	// Returns empty string if validation passes
	Validate(value string) string
	// Name returns the name of the validator for error reporting
	Name() string
}

// validators is a slice of Validator
type validators []Validator

// Validate applies all validators and returns the first error message
// Returns empty string if all validations pass
func (vs validators) Validate(value string) (string, string) {
	for _, v := range vs {
		if msg := v.Validate(value); msg != "" {
			return v.Name(), msg
		}
	}
	return "", ""
}

// =============================================================================
// Basic Validators
// =============================================================================

// requiredValidator validates that a value is not empty
type requiredValidator struct{}

// newRequiredValidator creates a new required validator
func newRequiredValidator() *requiredValidator {
	return &requiredValidator{}
}

// Validate checks if the value is not empty
func (v *requiredValidator) Validate(value string) string {
	if value == "" {
		return "value is required"
	}
	return ""
}

// Name returns the validator name
func (v *requiredValidator) Name() string {
	return requiredTagValue
}

// booleanValidator validates that a value is a boolean (true, false, 0, 1)
type booleanValidator struct{}

// newBooleanValidator creates a new boolean validator
func newBooleanValidator() *booleanValidator {
	return &booleanValidator{}
}

// Validate checks if the value is a valid boolean
func (v *booleanValidator) Validate(value string) string {
	if value == "true" || value == "false" || value == "0" || value == "1" {
		return ""
	}
	return "value must be a boolean (true, false, 0, or 1)"
}

// Name returns the validator name
func (v *booleanValidator) Name() string {
	return booleanTagValue
}

// alphaValidator validates that a value contains only ASCII alphabetic characters
type alphaValidator struct{}

// newAlphaValidator creates a new alpha validator
func newAlphaValidator() *alphaValidator {
	return &alphaValidator{}
}

// Validate checks if the value contains only alphabetic characters
func (v *alphaValidator) Validate(value string) string {
	for _, r := range value {
		if !isAlpha(r) {
			return "value must contain only alphabetic characters"
		}
	}
	return ""
}

// Name returns the validator name
func (v *alphaValidator) Name() string {
	return alphaTagValue
}

// isAlpha returns true if the rune is an ASCII alphabetic character
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// alphaUnicodeValidator validates that a value contains only unicode letters
type alphaUnicodeValidator struct{}

// newAlphaUnicodeValidator creates a new alphaUnicode validator
func newAlphaUnicodeValidator() *alphaUnicodeValidator {
	return &alphaUnicodeValidator{}
}

// Validate checks if the value contains only unicode letters
func (v *alphaUnicodeValidator) Validate(value string) string {
	for _, r := range value {
		if !unicode.IsLetter(r) {
			return "value must contain only unicode letters"
		}
	}
	return ""
}

// Name returns the validator name
func (v *alphaUnicodeValidator) Name() string {
	return alphaUnicodeTagValue
}

// alphaSpaceValidator validates that a value contains only alphabetic characters or spaces
type alphaSpaceValidator struct{}

// newAlphaSpaceValidator creates a new alphaSpace validator
func newAlphaSpaceValidator() *alphaSpaceValidator {
	return &alphaSpaceValidator{}
}

// Validate checks if the value contains only alphabetic characters or spaces
func (v *alphaSpaceValidator) Validate(value string) string {
	for _, r := range value {
		if !isAlpha(r) && r != ' ' {
			return "value must contain only alphabetic characters or spaces"
		}
	}
	return ""
}

// Name returns the validator name
func (v *alphaSpaceValidator) Name() string {
	return alphaSpaceTagValue
}

// numericValidator validates that a value is a valid integer
type numericValidator struct{}

// newNumericValidator creates a new numeric validator
func newNumericValidator() *numericValidator {
	return &numericValidator{}
}

// Validate checks if the value is a valid integer
func (v *numericValidator) Validate(value string) string {
	if value == "" {
		return ""
	}
	if _, err := strconv.Atoi(value); err != nil {
		return "value must be numeric"
	}
	return ""
}

// Name returns the validator name
func (v *numericValidator) Name() string {
	return numericTagValue
}

// numberValidator validates that a value is a valid number (integer or decimal)
type numberValidator struct{}

// newNumberValidator creates a new number validator
func newNumberValidator() *numberValidator {
	return &numberValidator{}
}

// Validate checks if the value is a valid number
func (v *numberValidator) Validate(value string) string {
	if !numberRegex.MatchString(value) {
		return errMsgValidNumber
	}
	return ""
}

// Name returns the validator name
func (v *numberValidator) Name() string {
	return numberTagValue
}

// alphanumericValidator validates that a value contains only ASCII alphanumeric characters
type alphanumericValidator struct{}

// newAlphanumericValidator creates a new alphanumeric validator
func newAlphanumericValidator() *alphanumericValidator {
	return &alphanumericValidator{}
}

// Validate checks if the value contains only alphanumeric characters
func (v *alphanumericValidator) Validate(value string) string {
	for _, r := range value {
		if !isAlpha(r) && !isNumeric(r) {
			return "value must contain only alphanumeric characters"
		}
	}
	return ""
}

// Name returns the validator name
func (v *alphanumericValidator) Name() string {
	return alphanumericTagValue
}

// isNumeric returns true if the rune is a numeric character
func isNumeric(r rune) bool {
	return r >= '0' && r <= '9'
}

// alphanumericUnicodeValidator validates unicode alphanumeric strings
type alphanumericUnicodeValidator struct{}

// newAlphanumericUnicodeValidator creates a new alphanumericUnicode validator
func newAlphanumericUnicodeValidator() *alphanumericUnicodeValidator {
	return &alphanumericUnicodeValidator{}
}

// Validate checks if the value contains only unicode letters or digits
func (v *alphanumericUnicodeValidator) Validate(value string) string {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "value must contain only unicode letters or digits"
		}
	}
	return ""
}

// Name returns the validator name
func (v *alphanumericUnicodeValidator) Name() string {
	return alphanumericUnicodeTagValue
}

// =============================================================================
// Comparison Validators
// =============================================================================

// equalValidator validates that a value equals the threshold
type equalValidator struct {
	threshold float64
}

// newEqualValidator creates a new equal validator
func newEqualValidator(threshold float64) *equalValidator {
	return &equalValidator{threshold: threshold}
}

// Validate checks if the value equals the threshold
func (v *equalValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f != v.threshold {
		return "value must equal " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *equalValidator) Name() string {
	return equalTagValue
}

// notEqualValidator validates that a value does not equal the threshold
type notEqualValidator struct {
	threshold float64
}

// newNotEqualValidator creates a new not equal validator
func newNotEqualValidator(threshold float64) *notEqualValidator {
	return &notEqualValidator{threshold: threshold}
}

// Validate checks if the value does not equal the threshold
func (v *notEqualValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f == v.threshold {
		return "value must not equal " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *notEqualValidator) Name() string {
	return notEqualTagValue
}

// greaterThanValidator validates that a value is greater than the threshold
type greaterThanValidator struct {
	threshold float64
}

// newGreaterThanValidator creates a new greater than validator
func newGreaterThanValidator(threshold float64) *greaterThanValidator {
	return &greaterThanValidator{threshold: threshold}
}

// Validate checks if the value is greater than the threshold
func (v *greaterThanValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f <= v.threshold {
		return "value must be greater than " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *greaterThanValidator) Name() string {
	return greaterThanTagValue
}

// greaterThanEqualValidator validates that a value is greater than or equal to the threshold
type greaterThanEqualValidator struct {
	threshold float64
}

// newGreaterThanEqualValidator creates a new greater than or equal validator
func newGreaterThanEqualValidator(threshold float64) *greaterThanEqualValidator {
	return &greaterThanEqualValidator{threshold: threshold}
}

// Validate checks if the value is greater than or equal to the threshold
func (v *greaterThanEqualValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f < v.threshold {
		return "value must be greater than or equal to " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *greaterThanEqualValidator) Name() string {
	return greaterThanEqualTagValue
}

// lessThanValidator validates that a value is less than the threshold
type lessThanValidator struct {
	threshold float64
}

// newLessThanValidator creates a new less than validator
func newLessThanValidator(threshold float64) *lessThanValidator {
	return &lessThanValidator{threshold: threshold}
}

// Validate checks if the value is less than the threshold
func (v *lessThanValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f >= v.threshold {
		return "value must be less than " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *lessThanValidator) Name() string {
	return lessThanTagValue
}

// lessThanEqualValidator validates that a value is less than or equal to the threshold
type lessThanEqualValidator struct {
	threshold float64
}

// newLessThanEqualValidator creates a new less than or equal validator
func newLessThanEqualValidator(threshold float64) *lessThanEqualValidator {
	return &lessThanEqualValidator{threshold: threshold}
}

// Validate checks if the value is less than or equal to the threshold
func (v *lessThanEqualValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f > v.threshold {
		return "value must be less than or equal to " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *lessThanEqualValidator) Name() string {
	return lessThanEqualTagValue
}

// minValidator validates that a value is at least the minimum
type minValidator struct {
	threshold float64
}

// newMinValidator creates a new min validator
func newMinValidator(threshold float64) *minValidator {
	return &minValidator{threshold: threshold}
}

// Validate checks if the value is at least the minimum
func (v *minValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f < v.threshold {
		return "value must be at least " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *minValidator) Name() string {
	return minTagValue
}

// maxValidator validates that a value is at most the maximum
type maxValidator struct {
	threshold float64
}

// newMaxValidator creates a new max validator
func newMaxValidator(threshold float64) *maxValidator {
	return &maxValidator{threshold: threshold}
}

// Validate checks if the value is at most the maximum
func (v *maxValidator) Validate(value string) string {
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return errMsgValidNumber
	}
	if f > v.threshold {
		return "value must be at most " + strconv.FormatFloat(v.threshold, 'f', -1, 64)
	}
	return ""
}

// Name returns the validator name
func (v *maxValidator) Name() string {
	return maxTagValue
}

// lengthValidator validates that a value has exactly the specified length
type lengthValidator struct {
	length int
}

// newLengthValidator creates a new length validator
func newLengthValidator(length int) *lengthValidator {
	return &lengthValidator{length: length}
}

// Validate checks if the value has exactly the specified length (grapheme clusters)
func (v *lengthValidator) Validate(value string) string {
	count := utf8.RuneCountInString(value)
	if count != v.length {
		return "value must have exactly " + strconv.Itoa(v.length) + " characters"
	}
	return ""
}

// Name returns the validator name
func (v *lengthValidator) Name() string {
	return lengthTagValue
}

// =============================================================================
// String Validators
// =============================================================================

// oneOfValidator validates that a value is one of the allowed values
type oneOfValidator struct {
	allowed []string
}

// newOneOfValidator creates a new oneOf validator
func newOneOfValidator(allowed []string) *oneOfValidator {
	return &oneOfValidator{allowed: allowed}
}

// Validate checks if the value is one of the allowed values
func (v *oneOfValidator) Validate(value string) string {
	for _, s := range v.allowed {
		if value == s {
			return ""
		}
	}
	return "value must be one of: " + strings.Join(v.allowed, ", ")
}

// Name returns the validator name
func (v *oneOfValidator) Name() string {
	return oneOfTagValue
}

// lowercaseValidator validates that a value is all lowercase
type lowercaseValidator struct{}

// newLowercaseValidator creates a new lowercase validator
func newLowercaseValidator() *lowercaseValidator {
	return &lowercaseValidator{}
}

// Validate checks if the value is all lowercase
func (v *lowercaseValidator) Validate(value string) string {
	if value != strings.ToLower(value) {
		return "value must be lowercase"
	}
	return ""
}

// Name returns the validator name
func (v *lowercaseValidator) Name() string {
	return lowercaseValidatorTagValue
}

// uppercaseValidator validates that a value is all uppercase
type uppercaseValidator struct{}

// newUppercaseValidator creates a new uppercase validator
func newUppercaseValidator() *uppercaseValidator {
	return &uppercaseValidator{}
}

// Validate checks if the value is all uppercase
func (v *uppercaseValidator) Validate(value string) string {
	if value != strings.ToUpper(value) {
		return "value must be uppercase"
	}
	return ""
}

// Name returns the validator name
func (v *uppercaseValidator) Name() string {
	return uppercaseValidatorTagValue
}

// asciiValidator validates that a value contains only ASCII characters
type asciiValidator struct{}

// newASCIIValidator creates a new ASCII validator
func newASCIIValidator() *asciiValidator {
	return &asciiValidator{}
}

// Validate checks if the value contains only ASCII characters
func (v *asciiValidator) Validate(value string) string {
	const maxASCII = 127
	for _, r := range value {
		if r > maxASCII {
			return "value must contain only ASCII characters"
		}
	}
	return ""
}

// Name returns the validator name
func (v *asciiValidator) Name() string {
	return asciiTagValue
}

// printASCIIValidator validates that a value contains only printable ASCII characters
type printASCIIValidator struct{}

// newPrintASCIIValidator creates a new printable ASCII validator
func newPrintASCIIValidator() *printASCIIValidator {
	return &printASCIIValidator{}
}

// Validate checks if the value contains only printable ASCII characters (0x20-0x7E)
func (v *printASCIIValidator) Validate(value string) string {
	for _, r := range value {
		if r < 0x20 || r > 0x7e {
			return "value must contain only printable ASCII characters"
		}
	}
	return ""
}

// Name returns the validator name
func (v *printASCIIValidator) Name() string {
	return printASCIITagValue
}

// =============================================================================
// Format Validators
// =============================================================================

// emailValidator validates that a value is a valid email address
type emailValidator struct{}

// newEmailValidator creates a new email validator
func newEmailValidator() *emailValidator {
	return &emailValidator{}
}

// Validate checks if the value is a valid email address
func (v *emailValidator) Validate(value string) string {
	if !emailRegex.MatchString(value) {
		return "value must be a valid email address"
	}
	return ""
}

// Name returns the validator name
func (v *emailValidator) Name() string {
	return emailTagValue
}

// uriValidator validates that a value is a valid URI
type uriValidator struct{}

// newURIValidator creates a new URI validator
func newURIValidator() *uriValidator {
	return &uriValidator{}
}

// Validate checks if the value is a valid URI
func (v *uriValidator) Validate(value string) string {
	if value == "" {
		return "value must be a valid URI"
	}

	// Strip fragment before parsing
	s := value
	if i := strings.Index(s, "#"); i > -1 {
		s = s[:i]
	}

	if _, err := url.ParseRequestURI(s); err != nil {
		return "value must be a valid URI"
	}
	return ""
}

// Name returns the validator name
func (v *uriValidator) Name() string {
	return uriTagValue
}

// urlValidator validates that a value is a valid URL
type urlValidator struct{}

// newURLValidator creates a new URL validator
func newURLValidator() *urlValidator {
	return &urlValidator{}
}

// Validate checks if the value is a valid URL
func (v *urlValidator) Validate(value string) string {
	if value == "" {
		return errMsgValidURL
	}

	parsed, err := url.Parse(strings.ToLower(value))
	if err != nil || parsed.Scheme == "" {
		return errMsgValidURL
	}

	isFileScheme := parsed.Scheme == fileScheme
	if (isFileScheme && (parsed.Path == "" || parsed.Path == "/")) ||
		(!isFileScheme && parsed.Host == "" && parsed.Fragment == "" && parsed.Opaque == "") {
		return errMsgValidURL
	}
	return ""
}

// Name returns the validator name
func (v *urlValidator) Name() string {
	return urlTagValue
}

// httpURLValidator validates that a value is a valid HTTP or HTTPS URL
type httpURLValidator struct{}

// newHTTPURLValidator creates a new HTTP URL validator
func newHTTPURLValidator() *httpURLValidator {
	return &httpURLValidator{}
}

// Validate checks if the value is a valid HTTP or HTTPS URL
func (v *httpURLValidator) Validate(value string) string {
	parsed, err := url.Parse(strings.ToLower(value))
	if err != nil || parsed.Host == "" {
		return "value must be a valid HTTP URL"
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "value must be a valid HTTP URL"
	}
	return ""
}

// Name returns the validator name
func (v *httpURLValidator) Name() string {
	return httpURLTagValue
}

// httpsURLValidator validates that a value is a valid HTTPS URL
type httpsURLValidator struct{}

// newHTTPSURLValidator creates a new HTTPS URL validator
func newHTTPSURLValidator() *httpsURLValidator {
	return &httpsURLValidator{}
}

// Validate checks if the value is a valid HTTPS URL
func (v *httpsURLValidator) Validate(value string) string {
	parsed, err := url.Parse(strings.ToLower(value))
	if err != nil || parsed.Host == "" || parsed.Scheme != "https" {
		return "value must be a valid HTTPS URL"
	}
	return ""
}

// Name returns the validator name
func (v *httpsURLValidator) Name() string {
	return httpsURLTagValue
}

// urlEncodedValidator validates that a value is URL encoded
type urlEncodedValidator struct{}

// newURLEncodedValidator creates a new URL encoded validator
func newURLEncodedValidator() *urlEncodedValidator {
	return &urlEncodedValidator{}
}

// Validate checks if the value is properly URL encoded
func (v *urlEncodedValidator) Validate(value string) string {
	if !urlEncodedRegex.MatchString(value) {
		return "value must be URL encoded"
	}
	return ""
}

// Name returns the validator name
func (v *urlEncodedValidator) Name() string {
	return urlEncodedTagValue
}

// dataURIValidator validates that a value is a valid data URI
type dataURIValidator struct{}

// newDataURIValidator creates a new data URI validator
func newDataURIValidator() *dataURIValidator {
	return &dataURIValidator{}
}

// Validate checks if the value is a valid data URI
func (v *dataURIValidator) Validate(value string) string {
	if !dataURIRegex.MatchString(value) {
		return errMsgValidDataURI
	}

	parts := strings.SplitN(value, ",", 2)
	if len(parts) != 2 {
		return errMsgValidDataURI
	}

	if _, err := base64.StdEncoding.DecodeString(parts[1]); err != nil {
		return errMsgValidDataURI
	}
	return ""
}

// Name returns the validator name
func (v *dataURIValidator) Name() string {
	return dataURITagValue
}

// =============================================================================
// Network Validators
// =============================================================================

// ipAddrValidator validates that a value is a valid IP address (IPv4 or IPv6)
type ipAddrValidator struct{}

// newIPAddrValidator creates a new IP address validator
func newIPAddrValidator() *ipAddrValidator {
	return &ipAddrValidator{}
}

// Validate checks if the value is a valid IP address
func (v *ipAddrValidator) Validate(value string) string {
	if value == "" || net.ParseIP(value) == nil {
		return "value must be a valid IP address"
	}
	return ""
}

// Name returns the validator name
func (v *ipAddrValidator) Name() string {
	return ipAddrTagValue
}

// ip4AddrValidator validates that a value is a valid IPv4 address
type ip4AddrValidator struct{}

// newIP4AddrValidator creates a new IPv4 address validator
func newIP4AddrValidator() *ip4AddrValidator {
	return &ip4AddrValidator{}
}

// Validate checks if the value is a valid IPv4 address
func (v *ip4AddrValidator) Validate(value string) string {
	if value == "" {
		return "value must be a valid IPv4 address"
	}
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		return "value must be a valid IPv4 address"
	}
	return ""
}

// Name returns the validator name
func (v *ip4AddrValidator) Name() string {
	return ip4AddrTagValue
}

// ip6AddrValidator validates that a value is a valid IPv6 address
type ip6AddrValidator struct{}

// newIP6AddrValidator creates a new IPv6 address validator
func newIP6AddrValidator() *ip6AddrValidator {
	return &ip6AddrValidator{}
}

// Validate checks if the value is a valid IPv6 address
func (v *ip6AddrValidator) Validate(value string) string {
	if value == "" {
		return "value must be a valid IPv6 address"
	}
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() != nil {
		return "value must be a valid IPv6 address"
	}
	return ""
}

// Name returns the validator name
func (v *ip6AddrValidator) Name() string {
	return ip6AddrTagValue
}

// cidrValidator validates that a value is a valid CIDR notation
type cidrValidator struct{}

// newCIDRValidator creates a new CIDR validator
func newCIDRValidator() *cidrValidator {
	return &cidrValidator{}
}

// Validate checks if the value is a valid CIDR notation
func (v *cidrValidator) Validate(value string) string {
	if value == "" {
		return "value must be a valid CIDR"
	}
	_, _, err := net.ParseCIDR(value)
	if err != nil {
		return "value must be a valid CIDR"
	}
	return ""
}

// Name returns the validator name
func (v *cidrValidator) Name() string {
	return cidrTagValue
}

// cidrv4Validator validates that a value is a valid IPv4 CIDR notation
type cidrv4Validator struct{}

// newCIDRv4Validator creates a new IPv4 CIDR validator
func newCIDRv4Validator() *cidrv4Validator {
	return &cidrv4Validator{}
}

// Validate checks if the value is a valid IPv4 CIDR notation
func (v *cidrv4Validator) Validate(value string) string {
	if value == "" {
		return "value must be a valid IPv4 CIDR"
	}
	ip, _, err := net.ParseCIDR(value)
	if err != nil || ip.To4() == nil {
		return "value must be a valid IPv4 CIDR"
	}
	return ""
}

// Name returns the validator name
func (v *cidrv4Validator) Name() string {
	return cidrv4TagValue
}

// cidrv6Validator validates that a value is a valid IPv6 CIDR notation
type cidrv6Validator struct{}

// newCIDRv6Validator creates a new IPv6 CIDR validator
func newCIDRv6Validator() *cidrv6Validator {
	return &cidrv6Validator{}
}

// Validate checks if the value is a valid IPv6 CIDR notation
func (v *cidrv6Validator) Validate(value string) string {
	if value == "" {
		return "value must be a valid IPv6 CIDR"
	}
	ip, _, err := net.ParseCIDR(value)
	if err != nil || ip.To4() != nil {
		return "value must be a valid IPv6 CIDR"
	}
	return ""
}

// Name returns the validator name
func (v *cidrv6Validator) Name() string {
	return cidrv6TagValue
}

// =============================================================================
// Identifier Validators
// =============================================================================

// uuidValidator validates that a value is a valid UUID
type uuidValidator struct{}

// newUUIDValidator creates a new UUID validator
func newUUIDValidator() *uuidValidator {
	return &uuidValidator{}
}

// Validate checks if the value is a valid UUID
func (v *uuidValidator) Validate(value string) string {
	if !uuidRegex.MatchString(value) {
		return "value must be a valid UUID"
	}
	return ""
}

// Name returns the validator name
func (v *uuidValidator) Name() string {
	return uuidTagValue
}

// fqdnValidator validates that a value is a valid fully qualified domain name
type fqdnValidator struct{}

// newFQDNValidator creates a new FQDN validator
func newFQDNValidator() *fqdnValidator {
	return &fqdnValidator{}
}

// Validate checks if the value is a valid FQDN
func (v *fqdnValidator) Validate(value string) string {
	if strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return errMsgValidFQDN
	}

	labels := strings.Split(value, ".")
	if len(labels) < 2 {
		return errMsgValidFQDN
	}

	totalLen := 0
	for _, label := range labels {
		totalLen += len(label) + 1
		if !fqdnLabelRegex.MatchString(label) {
			return errMsgValidFQDN
		}
	}

	if totalLen-1 > 253 {
		return errMsgValidFQDN
	}
	return ""
}

// Name returns the validator name
func (v *fqdnValidator) Name() string {
	return fqdnTagValue
}

// hostnameValidator validates that a value is a valid hostname (RFC 952)
type hostnameValidator struct{}

// newHostnameValidator creates a new hostname validator
func newHostnameValidator() *hostnameValidator {
	return &hostnameValidator{}
}

// Validate checks if the value is a valid hostname (RFC 952)
func (v *hostnameValidator) Validate(value string) string {
	return validateHostnameWithRegex(value, hostnameRFC952LabelRegex, "value must be a valid hostname")
}

// Name returns the validator name
func (v *hostnameValidator) Name() string {
	return hostnameTagValue
}

// hostnameRFC1123Validator validates that a value is a valid hostname (RFC 1123)
type hostnameRFC1123Validator struct{}

// newHostnameRFC1123Validator creates a new hostname RFC 1123 validator
func newHostnameRFC1123Validator() *hostnameRFC1123Validator {
	return &hostnameRFC1123Validator{}
}

// Validate checks if the value is a valid hostname (RFC 1123)
func (v *hostnameRFC1123Validator) Validate(value string) string {
	return validateHostnameWithRegex(value, hostnameRFC1123LabelRegex, "value must be a valid hostname (RFC 1123)")
}

// Name returns the validator name
func (v *hostnameRFC1123Validator) Name() string {
	return hostnameRFC1123TagValue
}

// validateHostnameWithRegex validates a hostname with the given label regex
func validateHostnameWithRegex(value string, labelRegex *regexp.Regexp, errMsg string) string {
	if strings.HasPrefix(value, ".") || strings.HasSuffix(value, ".") {
		return errMsg
	}

	labels := strings.Split(value, ".")
	if len(labels) < 1 {
		return errMsg
	}

	totalLen := 0
	for _, label := range labels {
		totalLen += len(label) + 1
		if !labelRegex.MatchString(label) {
			return errMsg
		}
	}

	if totalLen-1 > 253 {
		return errMsg
	}
	return ""
}

// hostnamePortValidator validates that a value is a valid hostname:port
type hostnamePortValidator struct{}

// newHostnamePortValidator creates a new hostname:port validator
func newHostnamePortValidator() *hostnamePortValidator {
	return &hostnamePortValidator{}
}

// Validate checks if the value is a valid hostname:port
func (v *hostnamePortValidator) Validate(value string) string {
	host, portStr, err := net.SplitHostPort(value)
	if err != nil {
		return errMsgValidHostnamePort
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return errMsgValidHostnamePort
	}

	// Check for IPv6 in brackets
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
			return ""
		}
		return errMsgValidHostnamePort
	}

	// Check if it's an IP address
	if ip := net.ParseIP(host); ip != nil {
		return ""
	}

	// Validate as hostname
	if validateHostnameWithRegex(host, hostnameRFC1123LabelRegex, "") != "" {
		return errMsgValidHostnamePort
	}
	return ""
}

// Name returns the validator name
func (v *hostnamePortValidator) Name() string {
	return hostnamePortTagValue
}

// =============================================================================
// String Content Validators
// =============================================================================

// startsWithValidator validates that a value starts with the prefix
type startsWithValidator struct {
	prefix string
}

// newStartsWithValidator creates a new startsWith validator
func newStartsWithValidator(prefix string) *startsWithValidator {
	return &startsWithValidator{prefix: prefix}
}

// Validate checks if the value starts with the prefix
func (v *startsWithValidator) Validate(value string) string {
	if !strings.HasPrefix(value, v.prefix) {
		return "value must start with '" + v.prefix + "'"
	}
	return ""
}

// Name returns the validator name
func (v *startsWithValidator) Name() string {
	return startsWithTagValue
}

// startsNotWithValidator validates that a value does not start with the prefix
type startsNotWithValidator struct {
	prefix string
}

// newStartsNotWithValidator creates a new startsNotWith validator
func newStartsNotWithValidator(prefix string) *startsNotWithValidator {
	return &startsNotWithValidator{prefix: prefix}
}

// Validate checks if the value does not start with the prefix
func (v *startsNotWithValidator) Validate(value string) string {
	if strings.HasPrefix(value, v.prefix) {
		return "value must not start with '" + v.prefix + "'"
	}
	return ""
}

// Name returns the validator name
func (v *startsNotWithValidator) Name() string {
	return startsNotWithTagValue
}

// endsWithValidator validates that a value ends with the suffix
type endsWithValidator struct {
	suffix string
}

// newEndsWithValidator creates a new endsWith validator
func newEndsWithValidator(suffix string) *endsWithValidator {
	return &endsWithValidator{suffix: suffix}
}

// Validate checks if the value ends with the suffix
func (v *endsWithValidator) Validate(value string) string {
	if !strings.HasSuffix(value, v.suffix) {
		return "value must end with '" + v.suffix + "'"
	}
	return ""
}

// Name returns the validator name
func (v *endsWithValidator) Name() string {
	return endsWithTagValue
}

// endsNotWithValidator validates that a value does not end with the suffix
type endsNotWithValidator struct {
	suffix string
}

// newEndsNotWithValidator creates a new endsNotWith validator
func newEndsNotWithValidator(suffix string) *endsNotWithValidator {
	return &endsNotWithValidator{suffix: suffix}
}

// Validate checks if the value does not end with the suffix
func (v *endsNotWithValidator) Validate(value string) string {
	if strings.HasSuffix(value, v.suffix) {
		return "value must not end with '" + v.suffix + "'"
	}
	return ""
}

// Name returns the validator name
func (v *endsNotWithValidator) Name() string {
	return endsNotWithTagValue
}

// containsValidator validates that a value contains the substring
type containsValidator struct {
	substr string
}

// newContainsValidator creates a new contains validator
func newContainsValidator(substr string) *containsValidator {
	return &containsValidator{substr: substr}
}

// Validate checks if the value contains the substring
func (v *containsValidator) Validate(value string) string {
	if !strings.Contains(value, v.substr) {
		return "value must contain '" + v.substr + "'"
	}
	return ""
}

// Name returns the validator name
func (v *containsValidator) Name() string {
	return containsTagValue
}

// containsAnyValidator validates that a value contains any of the substrings
type containsAnyValidator struct {
	substrs []string
}

// newContainsAnyValidator creates a new containsAny validator
func newContainsAnyValidator(substrs []string) *containsAnyValidator {
	return &containsAnyValidator{substrs: substrs}
}

// Validate checks if the value contains any of the substrings
func (v *containsAnyValidator) Validate(value string) string {
	for _, s := range v.substrs {
		if strings.Contains(value, s) {
			return ""
		}
	}
	return "value must contain one of: " + strings.Join(v.substrs, ", ")
}

// Name returns the validator name
func (v *containsAnyValidator) Name() string {
	return containsAnyTagValue
}

// containsRuneValidator validates that a value contains the rune
type containsRuneValidator struct {
	r rune
}

// newContainsRuneValidator creates a new containsRune validator
func newContainsRuneValidator(r rune) *containsRuneValidator {
	return &containsRuneValidator{r: r}
}

// Validate checks if the value contains the rune
func (v *containsRuneValidator) Validate(value string) string {
	if !strings.ContainsRune(value, v.r) {
		return "value must contain character '" + string(v.r) + "'"
	}
	return ""
}

// Name returns the validator name
func (v *containsRuneValidator) Name() string {
	return containsRuneTagValue
}

// =============================================================================
// Exclusion Validators
// =============================================================================

// excludesValidator validates that a value does not contain the substring
type excludesValidator struct {
	substr string
}

// newExcludesValidator creates a new excludes validator
func newExcludesValidator(substr string) *excludesValidator {
	return &excludesValidator{substr: substr}
}

// Validate checks if the value does not contain the substring
func (v *excludesValidator) Validate(value string) string {
	if strings.Contains(value, v.substr) {
		return "value must not contain '" + v.substr + "'"
	}
	return ""
}

// Name returns the validator name
func (v *excludesValidator) Name() string {
	return excludesTagValue
}

// excludesAllValidator validates that a value does not contain any of the runes
type excludesAllValidator struct {
	chars string
}

// newExcludesAllValidator creates a new excludesAll validator
func newExcludesAllValidator(chars string) *excludesAllValidator {
	return &excludesAllValidator{chars: chars}
}

// Validate checks if the value does not contain any of the specified characters
func (v *excludesAllValidator) Validate(value string) string {
	if value == "" || v.chars == "" {
		return ""
	}
	if strings.ContainsAny(value, v.chars) {
		return "value must not contain any of: " + v.chars
	}
	return ""
}

// Name returns the validator name
func (v *excludesAllValidator) Name() string {
	return excludesAllTagValue
}

// excludesRuneValidator validates that a value does not contain the rune
type excludesRuneValidator struct {
	r rune
}

// newExcludesRuneValidator creates a new excludesRune validator
func newExcludesRuneValidator(r rune) *excludesRuneValidator {
	return &excludesRuneValidator{r: r}
}

// Validate checks if the value does not contain the rune
func (v *excludesRuneValidator) Validate(value string) string {
	if strings.ContainsRune(value, v.r) {
		return "value must not contain character '" + string(v.r) + "'"
	}
	return ""
}

// Name returns the validator name
func (v *excludesRuneValidator) Name() string {
	return excludesRuneTagValue
}

// =============================================================================
// Misc Validators
// =============================================================================

// multibyteValidator validates that a value contains multibyte characters
type multibyteValidator struct{}

// newMultibyteValidator creates a new multibyte validator
func newMultibyteValidator() *multibyteValidator {
	return &multibyteValidator{}
}

// Validate checks if the value contains at least one multibyte character
func (v *multibyteValidator) Validate(value string) string {
	if value == "" || utf8.RuneCountInString(value) == len(value) {
		return "value must contain multibyte characters"
	}
	return ""
}

// Name returns the validator name
func (v *multibyteValidator) Name() string {
	return multibyteTagValue
}

// equalIgnoreCaseValidator validates that a value equals the expected value (case insensitive)
type equalIgnoreCaseValidator struct {
	expected string
}

// newEqualIgnoreCaseValidator creates a new equalIgnoreCase validator
func newEqualIgnoreCaseValidator(expected string) *equalIgnoreCaseValidator {
	return &equalIgnoreCaseValidator{expected: expected}
}

// Validate checks if the value equals the expected value (case insensitive)
func (v *equalIgnoreCaseValidator) Validate(value string) string {
	if !strings.EqualFold(value, v.expected) {
		return "value must equal '" + v.expected + "' (case insensitive)"
	}
	return ""
}

// Name returns the validator name
func (v *equalIgnoreCaseValidator) Name() string {
	return equalIgnoreCaseTagValue
}

// notEqualIgnoreCaseValidator validates that a value does not equal the expected value (case insensitive)
type notEqualIgnoreCaseValidator struct {
	expected string
}

// newNotEqualIgnoreCaseValidator creates a new notEqualIgnoreCase validator
func newNotEqualIgnoreCaseValidator(expected string) *notEqualIgnoreCaseValidator {
	return &notEqualIgnoreCaseValidator{expected: expected}
}

// Validate checks if the value does not equal the expected value (case insensitive)
func (v *notEqualIgnoreCaseValidator) Validate(value string) string {
	if strings.EqualFold(value, v.expected) {
		return "value must not equal '" + v.expected + "' (case insensitive)"
	}
	return ""
}

// Name returns the validator name
func (v *notEqualIgnoreCaseValidator) Name() string {
	return notEqualIgnoreCaseTagValue
}
