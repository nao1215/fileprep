package fileprep_test

import (
	"fmt"
	"io"
	"strings"

	"github.com/nao1215/fileprep"
)

// User represents a user record with preprocessing and validation
type User struct {
	Name  string `prep:"trim" validate:"required"`
	Email string `prep:"trim,lowercase" validate:"required"`
	Age   string
}

func Example() {
	// Sample CSV data with whitespace that needs trimming
	csvData := `name,email,age
  John Doe  ,JOHN@EXAMPLE.COM,30
Jane Smith,jane@example.com,25
`

	// Create a processor for CSV files
	processor := fileprep.NewProcessor(fileprep.FileTypeCSV)

	// Prepare a slice to hold the parsed records
	var users []User

	// Process the data
	reader, result, err := processor.Process(strings.NewReader(csvData), &users)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print processing results
	fmt.Printf("Processed %d rows, %d valid\n", result.RowCount, result.ValidRowCount)

	// Print parsed records (with preprocessing applied)
	for i, user := range users {
		fmt.Printf("User %d: Name=%q, Email=%q\n", i+1, user.Name, user.Email)
	}

	// The reader can be passed to filesql
	// For demonstration, we just read and show the preprocessed output
	output, err := io.ReadAll(reader)
	if err != nil {
		fmt.Printf("Error reading output: %v\n", err)
		return
	}
	fmt.Printf("Output for filesql:\n%s", output)

	// Output:
	// Processed 2 rows, 2 valid
	// User 1: Name="John Doe", Email="john@example.com"
	// User 2: Name="Jane Smith", Email="jane@example.com"
	// Output for filesql:
	// name,email,age
	// John Doe,john@example.com,30
	// Jane Smith,jane@example.com,25
}

func Example_validation() {
	// CSV data with validation error (empty required name)
	csvData := `name,email,age
,john@example.com,30
Jane,jane@example.com,25
`

	processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
	var users []User

	_, result, err := processor.Process(strings.NewReader(csvData), &users)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Check for validation errors
	if result.HasErrors() {
		fmt.Printf("Found %d validation errors:\n", len(result.Errors))
		for _, e := range result.ValidationErrors() {
			fmt.Printf("  Row %d, Column %q: %s\n", e.Row, e.Column, e.Message)
		}
	}

	fmt.Printf("Valid rows: %d/%d\n", result.ValidRowCount, result.RowCount)

	// Output:
	// Found 1 validation errors:
	//   Row 1, Column "name": value is required
	// Valid rows: 1/2
}

// Example_complexPrepAndValidation demonstrates comprehensive preprocessing and validation
// using multiple prep tags and validation rules for realistic data processing scenarios.
func Example_complexPrepAndValidation() {
	// Product represents a product record with comprehensive preprocessing and validation
	type Product struct {
		// Name: trim whitespace, require non-empty
		Name string `prep:"trim" validate:"required"`
		// SKU: trim, uppercase, require non-empty and alphanumeric
		SKU string `prep:"trim,uppercase" validate:"required,alphanumeric"`
		// Price: trim, set default if empty, validate as number (int or decimal)
		Price string `prep:"trim,default=0.00" validate:"number"`
		// Quantity: trim, coerce to int, validate as integer
		Quantity string `prep:"trim,coerce=int" validate:"numeric"`
		// Category: trim, lowercase, collapse multiple spaces
		Category string `prep:"trim,lowercase,collapse_space"`
		// Description: trim, strip HTML, truncate to 200 chars
		Description string `prep:"trim,strip_html,truncate=200"`
		// URL: trim, fix scheme (add https:// if missing)
		URL string `prep:"trim,fix_scheme=https"`
		// Tags: trim, replace semicolons with commas
		Tags string `prep:"trim,replace=;:,"`
	}

	// Sample CSV with various data quality issues
	csvData := `name,sku,price,quantity,category,description,url,tags
  Widget Pro  ,ABC123,19.99,100,  Electronics   ,<p>A <b>great</b> product!</p>,example.com/widget,tag1;tag2;tag3
 Gadget Plus ,DEF456,29.99,50,  home  goods  ,Simple description,https://example.com/gadget,electronics;sale
  ,ABC@#$,not_a_number,5,test,desc,url,tags
`

	processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
	var products []Product

	reader, result, err := processor.Process(strings.NewReader(csvData), &products)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Show processing summary
	fmt.Printf("Processing Summary:\n")
	fmt.Printf("  Total rows: %d\n", result.RowCount)
	fmt.Printf("  Valid rows: %d\n", result.ValidRowCount)
	fmt.Printf("  Invalid rows: %d\n", result.InvalidRowCount())

	// Show validation errors
	if result.HasErrors() {
		fmt.Printf("\nValidation Errors:\n")
		for _, ve := range result.ValidationErrors() {
			fmt.Printf("  Row %d, Field %q: %s\n", ve.Row, ve.Field, ve.Message)
		}
	}

	// Show preprocessed records
	fmt.Printf("\nPreprocessed Records:\n")
	for i, p := range products {
		if p.Name != "" { // Skip invalid rows for display
			fmt.Printf("  [%d] Name=%q, SKU=%q, Price=%q, Category=%q\n",
				i+1, p.Name, p.SKU, p.Price, p.Category)
		}
	}

	// Show that URL scheme was fixed
	fmt.Printf("\nURL Examples (scheme fixed):\n")
	for i, p := range products {
		if p.URL != "" && p.Name != "" {
			fmt.Printf("  [%d] %s\n", i+1, p.URL)
		}
	}

	// The reader can be used with filesql
	_, _ = io.ReadAll(reader) //nolint:errcheck // Example code - ignoring error

	// Output:
	// Processing Summary:
	//   Total rows: 3
	//   Valid rows: 2
	//   Invalid rows: 1
	//
	// Validation Errors:
	//   Row 3, Field "Name": value is required
	//   Row 3, Field "SKU": value must contain only alphanumeric characters
	//   Row 3, Field "Price": value must be a valid number
	//
	// Preprocessed Records:
	//   [1] Name="Widget Pro", SKU="ABC123", Price="19.99", Category="electronics"
	//   [2] Name="Gadget Plus", SKU="DEF456", Price="29.99", Category="home goods"
	//
	// URL Examples (scheme fixed):
	//   [1] https://example.com/widget
	//   [2] https://example.com/gadget
}

// Example_crossFieldValidation demonstrates validation rules that compare values
// between different fields, such as password confirmation matching.
func Example_crossFieldValidation() {
	// UserForm represents a user registration form with cross-field validation
	type UserForm struct {
		Username        string `prep:"trim,lowercase" validate:"required"`
		Password        string `validate:"required"`
		ConfirmPassword string `validate:"required,eqfield=Password"`
	}

	csvData := `username,password,confirm_password
  Alice  ,secret123,secret123
Bob,password1,wrongpass
`

	processor := fileprep.NewProcessor(fileprep.FileTypeCSV)
	var users []UserForm

	_, result, err := processor.Process(strings.NewReader(csvData), &users)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Valid: %d/%d\n", result.ValidRowCount, result.RowCount)

	if result.HasErrors() {
		fmt.Printf("Errors:\n")
		for _, ve := range result.ValidationErrors() {
			fmt.Printf("  Row %d, Field %q: %s\n", ve.Row, ve.Field, ve.Message)
		}
	}

	// Output:
	// Valid: 1/2
	// Errors:
	//   Row 2, Field "ConfirmPassword": value must equal field Password
}

// Example_detectFileType demonstrates automatic file type detection from file paths.
func Example_detectFileType() {
	files := []string{
		"data.csv",
		"data.csv.gz",
		"report.xlsx",
		"logs.tsv.bz2",
		"events.parquet",
		"access.ltsv.zst",
	}

	for _, f := range files {
		ft := fileprep.DetectFileType(f)
		fmt.Printf("%s -> %s (compressed: %v)\n", f, ft, ft.IsCompressed())
	}

	// Output:
	// data.csv -> CSV (compressed: false)
	// data.csv.gz -> CSV (gzip) (compressed: true)
	// report.xlsx -> XLSX (compressed: false)
	// logs.tsv.bz2 -> TSV (bzip2) (compressed: true)
	// events.parquet -> Parquet (compressed: false)
	// access.ltsv.zst -> LTSV (zstd) (compressed: true)
}
