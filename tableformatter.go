package tableformatter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v2"
)

//Table represents a table including data and schema
type Table struct {
	Data   [][]interface{}
	Schema []SchemaField
}

var (
	DefaultDelimiter    = "|"
	DefaultTimeFormat   = "2006-01-02T15:04:05Z" //oddly enough this is how you specify a format
	DefaultFoldAtLength = 300
)

const (
	//TypeInt is printed as %d
	TypeInt = iota
	//TypeString is printed as %s
	TypeString = iota
	//TypeFloat is printed as %f
	TypeFloat = iota
	//TypeDateTime is printed as a string after parsing
	TypeDateTime = iota
	//TypeInterface is printed as %v
	TypeInterface = iota
	//TypeBool is printed as %v
	TypeBool = iota
)

//SchemaField defines a field in a table
type SchemaField struct {
	FieldName      string
	FieldType      int
	FieldSize      int
	FieldPrecision int
	FieldFormat    string
}

type lessFunc func(p1, p2 interface{}, field *SchemaField) bool

// MultiSorter implements the Sort interface, sorting the changes within.
type MultiSorter struct {
	data    [][]interface{}
	less    []lessFunc
	schema  []SchemaField
	indexes []int
}

// Sort sorts the argument slice according to the less functions passed to OrderedBy.
func (ms *MultiSorter) Sort(data [][]interface{}) {
	ms.data = data
	sort.Sort(ms)
}

// Len is part of sort.Interface.
func (ms *MultiSorter) Len() int {
	return len(ms.data)
}

// Swap is part of sort.Interface.
func (ms *MultiSorter) Swap(i, j int) {
	ms.data[i], ms.data[j] = ms.data[j], ms.data[i]
}

// Less is part of sort.Interface. It is implemented by looping along the
// less functions until it finds a comparison that discriminates between
// the two items (one is less than the other). Note that it can call the
// less functions twice per call. We could change the functions to return
// -1, 0, 1 and reduce the number of calls for greater efficiency: an
// exercise for the reader.
func (ms *MultiSorter) Less(i, j int) bool {
	p, q := ms.data[i], ms.data[j]
	// Try all but the last comparison.
	var k int
	for k = 0; k < len(ms.less)-1; k++ {
		less := ms.less[k]
		index := ms.indexes[k]
		switch {
		case less(p[index], q[index], &ms.schema[index]):
			// p < q, so we have a decision.
			return true
		case less(q[index], p[index], &ms.schema[index]):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}
	// All comparisons to here said "equal", so just return whatever
	// the final comparison reports.
	lastIndex := ms.indexes[k]
	return ms.less[k](p[lastIndex], q[lastIndex], &ms.schema[lastIndex])
}

//TableSorter a multisorter for a table
func TableSorter(schema []SchemaField) *MultiSorter {
	return &MultiSorter{
		schema: schema,
	}
}

//OrderBy specifies the order
func (ms *MultiSorter) OrderBy(fieldNames ...string) *MultiSorter {

	ms.less = make([]lessFunc, len(fieldNames))
	ms.indexes = make([]int, len(fieldNames))

	for k, fn := range fieldNames {
		var field *SchemaField
		for i, f := range ms.schema {

			if f.FieldName == fn {
				field = &f
				ms.indexes[k] = i
				break
			}
		}

		if field == nil {
			fmt.Printf("could not find field with name %s\n", fn)
			return nil
		}

		switch field.FieldType {
		case TypeInt:
			ms.less[k] = func(a, b interface{}, field *SchemaField) bool {
				return a.(int) < b.(int)
			}
		case TypeString:
			ms.less[k] = func(a, b interface{}, field *SchemaField) bool {
				return a.(string) < b.(string)
			}
		case TypeFloat:
			ms.less[k] = func(a, b interface{}, field *SchemaField) bool {
				return a.(float64) < b.(float64)
			}
		case TypeDateTime:
			ms.less[k] = func(a, b interface{}, field *SchemaField) bool {

				layout := DefaultTimeFormat

				if field.FieldFormat != "" {
					layout = field.FieldFormat
				}

				ta, err := time.Parse(layout, a.(string))
				if err != nil {
					fmt.Printf("could not convert string %s to date time", a.(string))
					return false
				}

				tb, err := time.Parse(layout, b.(string))
				if err != nil {
					fmt.Printf("could not convert string %s to date time", b.(string))
					return false
				}

				return ta.Before(tb)
			}
		case TypeBool:
			ms.less[k] = func(a, b interface{}, field *SchemaField) bool {
				return a.(bool) != b.(bool)
			}
		default:
			fmt.Printf("could not find type %d", field.FieldType)
		}
	}

	return ms
}

//getTableHeader returns the row for header (all cells strings but of the length specified in the schema)
func getTableHeader(schema []SchemaField) string {
	var alteredSchema []SchemaField
	var header []interface{}

	for _, field := range schema {
		alteredSchema = append(alteredSchema, SchemaField{
			FieldType: TypeString,
			FieldSize: field.FieldSize,
		})
		header = append(header, field.FieldName)
	}
	return getTableRow(header, alteredSchema)
}

func emptyString(length int) string {
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteString(" ")
	}
	return sb.String()
}

//pad string with spaces up to certain len
func pad(s string, desired_len int) string {
	current_len := len(decolorize(s))
	l := desired_len - current_len
	if l <= 0 {
		return s
	}
	return s + emptyString(l)
}

//getTableRow returns the string for a row with the | delimiter
func getTableRow(row []interface{}, schema []SchemaField) string {
	//row[0] is the first cell row[1] second cell row[1][1] is the value of the second row of the second cell
	//this is to allow multi-line string cells
	var rowStr [][]string
	rowHeight := 1

	for i, field := range schema {
		switch field.FieldType {
		case TypeInt:
			rowStr = append(rowStr, []string{fmt.Sprintf(fmt.Sprintf(" %%-%dd", field.FieldSize), row[i].(int))})
		case TypeString:
			//escape %
			s := strings.ReplaceAll(row[i].(string), "%", "%%")
			splittedS := strings.Split(s, "\n")
			multiLineCell := []string{}
			for _, r := range splittedS {
				ds := " " + pad(r, field.FieldSize)
				multiLineCell = append(multiLineCell, ds)
			}
			if rowHeight < len(multiLineCell) {
				rowHeight = len(multiLineCell)
			}
			rowStr = append(rowStr, multiLineCell)

		case TypeFloat:
			rowStr = append(rowStr, []string{fmt.Sprintf(fmt.Sprintf(" %%-%d.%df", field.FieldSize, field.FieldPrecision), row[i].(float64))})
		default:
			rowStr = append(rowStr, []string{fmt.Sprintf(fmt.Sprintf(" %%-+%dv", field.FieldSize), row[i])})
		}
	}

	//for each cell fill it to rowHeight with empty strings of length equal to the largest
	for i, cell := range rowStr {
		//determine the max width
		maxWidth := 0
		for _, s := range cell {
			if len(s) > maxWidth {

				maxWidth = len(decolorize(s))
			}
		}
		newCell := []string{}
		//adjust sizes to all other fields by padding them with spaces
		for j := 0; j < len(cell); j++ {
			newCell = append(newCell, fmt.Sprintf(fmt.Sprintf("%%-%ds", maxWidth), cell[j]))
		}
		//fill remainder with empty strings
		for j := len(cell); j < rowHeight; j++ {
			newCell = append(newCell, emptyString(maxWidth))
		}
		//replace original cell
		rowStr[i] = newCell
	}

	var sb strings.Builder

	for y := 0; y < rowHeight; y++ {

		for x := 0; x < len(rowStr); x++ {
			sb.WriteString(DefaultDelimiter)
			sb.WriteString(rowStr[x][y])
		}
		sb.WriteString(DefaultDelimiter)
		if y < rowHeight-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

//removes all coloring characters
func decolorize(s string) string {
	r := regexp.MustCompile(`\x1b\[[0-9;]*[mG]`)
	return r.ReplaceAllLiteralString(s, "")
}

// GetCellSize calculates how wide a cell is by converting it to string and measuring it's size
func getCellSize(d interface{}, field *SchemaField) int {
	var s string
	switch field.FieldType {
	case TypeInt:
		s = fmt.Sprintf("%d", d.(int))
	case TypeString:
		s = decolorize(d.(string))
	case TypeFloat:
		s = fmt.Sprintf(fmt.Sprintf("%%.%df", field.FieldPrecision), d.(float64))
	default:
		s = decolorize(fmt.Sprintf("%v", d))

	}
	//if multi-line measure the widest string in array
	splittedS := strings.Split(s, "\n")
	maxW := 0
	for _, w := range splittedS {
		if maxW < len(w) {
			maxW = len(w)
		}
	}
	return maxW
}

//getRowSize returns the row size of a table
func getRowSize(data [][]interface{}, schema []SchemaField) int {
	rowSize := len(schema)
	size := 0
	for i := 0; i < rowSize; i++ {
		f := (schema)[i]
		size += getCellSize(data[0][i], &f)
	}
	return size
}

//AdjustFieldSizes expands field sizes to match the widest cell
func (t *Table) AdjustFieldSizes() {

	rowSize := len(t.Schema)
	for i := 0; i < rowSize; i++ {
		f := t.Schema[i]

		//iterate over the entire column
		rowCount := len(t.Data)

		maxLen := f.FieldSize

		if len(f.FieldName) > maxLen {
			maxLen = len(f.FieldName)
		}

		for k := 0; k < rowCount; k++ {
			cellSize := getCellSize(t.Data[k][i], &f)
			if cellSize > maxLen {
				maxLen = cellSize
			}
		}
		if maxLen > f.FieldSize {
			t.Schema[i].FieldSize = maxLen + 1 //we leave a little room to the right
		}
	}
}

//getTableDelimiter returns a delimiter row for the schema
func getTableDelimiter(schema []SchemaField) string {
	row := "+"
	for _, field := range schema {
		for i := 0; i < field.FieldSize+1; i++ {
			row += "-"
		}
		row += "+"
	}
	return row
}

//getTableAsString returns the string representation of a table.
func getTableAsString(data [][]interface{}, schema []SchemaField) string {
	var rows []string

	rows = append(rows, getTableDelimiter(schema))
	rows = append(rows, getTableHeader(schema))
	rows = append(rows, getTableDelimiter(schema))
	for _, row := range data {
		rows = append(rows, getTableRow(row, schema))
	}
	rows = append(rows, getTableDelimiter(schema))

	return strings.Join(rows, "\n") + "\n"
}

//getFoldedTableAsString returns the string representation of a table with the fields collapsed
func getFoldedTableAsString(data [][]interface{}, schema []SchemaField) (string, error) {

	newSchema := []SchemaField{
		{
			FieldName: "Values",
			FieldType: TypeString,
			FieldSize: 5,
		},
	}
	newData := [][]interface{}{}
	for _, row := range data {

		cell, err := getTableAsYAMLString([][]interface{}{row}, schema)
		if err != nil {
			return "", err
		}
		newData = append(newData, []interface{}{cell})

	}

	table := Table{newData, newSchema}
	table.AdjustFieldSizes()

	return getTableAsString(table.Data, table.Schema), nil
}

func printTableHeader(schema []SchemaField) {
	fmt.Println(getTableHeader(schema))
}

func printTableRow(row []interface{}, schema []SchemaField) {
	fmt.Println(getTableRow(row, schema))
}

func printTableDelimiter(schema []SchemaField) {
	fmt.Println(getTableDelimiter(schema))
}

func printTable(data [][]interface{}, schema []SchemaField) {
	fmt.Print(getTableAsString(data, schema))
}

//getTableAsYAMLString returns a yaml.Marshal string for the given data
func getTableAsYAMLString(data [][]interface{}, schema []SchemaField) (string, error) {

	dataAsMap := make([]interface{}, len(data))

	for k, row := range data {
		rowAsMap := make(map[string]interface{}, len(schema))
		for i, field := range schema {
			formattedFieldName := strcase.ToLowerCamel(strings.ToLower(field.FieldName))
			rowAsMap[formattedFieldName] = row[i]
		}
		dataAsMap[k] = rowAsMap
	}

	ret, err := yaml.Marshal(dataAsMap)
	if err != nil {
		return "", err
	}

	return decolorize(string(ret)), nil
}

//getTableAsJSONString returns a json.MarshalIndent string for the given data
func getTableAsJSONString(data [][]interface{}, schema []SchemaField) (string, error) {
	dataAsMap := make([]interface{}, len(data))

	for k, row := range data {
		rowAsMap := make(map[string]interface{}, len(schema))
		for i, field := range schema {
			rowAsMap[field.FieldName] = row[i]
		}
		dataAsMap[k] = rowAsMap
	}

	ret, err := json.MarshalIndent(dataAsMap, "", "\t")
	if err != nil {
		return "", err
	}

	return decolorize(string(ret)), nil
}

//getTableAsCSVString returns a table as a csv
func getTableAsCSVString(data [][]interface{}, schema []SchemaField) (string, error) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	csvWriter := csv.NewWriter(writer)

	rowStr := make([]string, len(schema))
	for i, field := range schema {
		rowStr[i] = field.FieldName
	}

	csvWriter.Write(rowStr)

	for _, row := range data {
		for i, field := range schema {
			switch field.FieldType {
			case TypeInt:
				rowStr[i] = fmt.Sprintf("%d", row[i].(int))
			case TypeString:
				rowStr[i] = row[i].(string)
			case TypeFloat:
				rowStr[i] = fmt.Sprintf("%f", row[i].(float64))
			case TypeInterface:
				rowStr[i] = fmt.Sprintf("%v", row[i])
			case TypeDateTime:
				rowStr[i] = row[i].(string)
			default:
				rowStr[i] = fmt.Sprintf("%v", row[i])
			}
		}
		csvWriter.Write(rowStr)
	}

	writer.Flush()
	csvWriter.Flush()

	return decolorize(buf.String()), nil
}

func truncateString(s string, length int) string {
	str := s
	if len(str) > 0 {
		return str[:length] + "..."
	}
	return ""
}

//RenderTableAsJSON renders the table as an array json objects
func (t *Table) RenderTableAsJSON() (string, error) {
	return t.RenderTableFoldable("", "", "json", DefaultFoldAtLength)
}

//RenderTableAsCSV renders the table as a csv
func (t *Table) RenderTableAsCSV() (string, error) {
	return t.RenderTableFoldable("", "", "csv", DefaultFoldAtLength)
}

//RenderTableAsYAML renders the table as a yaml object
func (t *Table) RenderTableAsYAML() (string, error) {
	return t.RenderTableFoldable("", "", "yaml", DefaultFoldAtLength)
}

//RenderTable renders a table object as a string
//supported formats: json, csv, yaml
func (t *Table) RenderTable(tableName string, topLine string, format string) (string, error) {
	return t.RenderTableFoldable(tableName, topLine, format, DefaultFoldAtLength)
}

//RenderTableFoldable renders a table object as a string
//supported formats: json, csv, yaml
//foldAtLength specifies at which row length to fold the
func (t *Table) RenderTableFoldable(tableName string, topLine string, format string, foldAtLength int) (string, error) {
	var sb strings.Builder

	switch format {
	case "json", "JSON":
		ret, err := getTableAsJSONString(t.Data, t.Schema)
		if err != nil {
			return "", err
		}
		sb.WriteString(ret)
	case "csv", "CSV":
		ret, err := getTableAsCSVString(t.Data, t.Schema)
		if err != nil {
			return "", err
		}
		sb.WriteString(ret)
	case "yaml", "YAML":
		ret, err := getTableAsYAMLString(t.Data, t.Schema)
		if err != nil {
			return "", err
		}
		sb.WriteString(ret)

	case "":
		if topLine != "" {
			sb.WriteString(fmt.Sprintf("%s\n", topLine))
		}

		t.AdjustFieldSizes()

		if len(t.Data) > 0 && getRowSize(t.Data, t.Schema) > foldAtLength {
			s, err := getFoldedTableAsString(t.Data, t.Schema)
			if err != nil {
				return "", err
			}
			sb.WriteString(s)
		} else {
			sb.WriteString(getTableAsString(t.Data, t.Schema))
		}

		sb.WriteString(fmt.Sprintf("Total: %d %s\n\n", len(t.Data), tableName))

	default:
		return "", fmt.Errorf("Invalid format '%s' given. Valid values are json, JSON, csv, CSV, yaml or YAML.", format)
	}

	return sb.String(), nil
}

//TransposeTable turns columns into rows. It assumes an uniform length table
func TransposeTable(t Table) Table {

	dataT := [][]interface{}{}

	if len(t.Data) == 0 {
		return t
	}

	tableLength := len(t.Data)
	rowLength := len(t.Data[0])

	for j := 0; j < rowLength; j++ {

		newRow := []interface{}{}

		for i := 0; i < tableLength; i++ {

			newRow = append(newRow, t.Data[i][j])
		}

		dataT = append(dataT, newRow)
	}
	newTable := Table{dataT, t.Schema}
	return newTable
}

//ConvertToStringTable converts all cells to string cells
func ConvertToStringTable(table Table) Table {
	dataS := [][]interface{}{}

	for _, row := range table.Data {
		newRow := []interface{}{}
		for _, v := range row {
			if v == nil {
				v = " "
			}
			newRow = append(newRow, fmt.Sprintf("%v", v))
		}
		dataS = append(dataS, newRow)
	}
	newTable := Table{
		Data:   dataS,
		Schema: table.Schema,
	}
	return newTable
}

//RenderTransposedTable renders the text format as a key-value table. json and csv formats remain the same as render table
//supported formats: json, csv, yaml
func (t *Table) RenderTransposedTable(tableName string, topLine string, format string) (string, error) {

	if format != "" {
		return t.RenderTable(tableName, topLine, format)
	}

	headerRow := []interface{}{}
	for _, s := range t.Schema {
		headerRow = append(headerRow, s.FieldName)
	}

	stringsTable := ConvertToStringTable(*t)

	newDataAsStrings := [][]interface{}{}
	newDataAsStrings = append(newDataAsStrings, headerRow)
	for _, row := range stringsTable.Data {
		newDataAsStrings = append(newDataAsStrings, row)
	}

	newSchema := []SchemaField{
		{
			FieldName: "KEY",
			FieldType: TypeString,
			FieldSize: 5,
		},
		{
			FieldName: "VALUE",
			FieldType: TypeString,
			FieldSize: 5,
		},
	}

	newTable := Table{newDataAsStrings, newSchema}
	tableTransposed := TransposeTable(newTable)

	return tableTransposed.RenderTable(tableName, topLine, format)

}

//RenderTransposedTableHumanReadable renders an object in a human readable way
func (t *Table) RenderTransposedTableHumanReadable(tableName string, topLine string) (string, error) {

	headerRow := []interface{}{}
	for _, s := range t.Schema {
		headerRow = append(headerRow, s.FieldName)
	}

	var sb strings.Builder
	for i, field := range t.Schema {
		sb.WriteString(fmt.Sprintf("%s: %v\n", field.FieldName, t.Data[0][i]))
	}

	return sb.String(), nil
}

//FieldNameFormatter is a formatter for fields
type FieldNameFormatter interface {
	Format(n string) string
}

//HumanReadableFormatter formats a field in the form "word word word"
type HumanReadableFormatter struct{}

//Format returns formatted string
func (o *HumanReadableFormatter) Format(s string) string {
	return strcase.ToDelimited(s, ' ')
}

//NewHumanReadableFormatter creates a new formatter
func NewHumanReadableFormatter() *HumanReadableFormatter { return &HumanReadableFormatter{} }

//PassThroughFormatter formats a field in the form "word word word"
type PassThroughFormatter struct{}

//Format returns formatted string
func (o *PassThroughFormatter) Format(s string) string {
	return s
}

//NewPassThroughFormatter passthriugh
func NewPassThroughFormatter() *PassThroughFormatter { return &PassThroughFormatter{} }

//StripPrefixFormatter strips a prefix from field names
type StripPrefixFormatter struct {
	Prefix string
}

//Format returns formatted string
func (o *StripPrefixFormatter) Format(s string) string {
	return strings.Title(strcase.ToDelimited(strings.TrimPrefix(s, o.Prefix), ' '))
}

//NewStripPrefixFormatter like HumanReadableFormatter but strips a prefix
func NewStripPrefixFormatter(prefix string) *StripPrefixFormatter {
	return &StripPrefixFormatter{Prefix: prefix}
}

//ObjectToTable converts an object into a table directly
//without having to manually build the schema and fields
func ObjectToTable(obj interface{}) (*Table, error) {
	return ObjectToTableWithFormatter(obj, NewHumanReadableFormatter())
}

//ObjectToTableWithFormatter converts an object into a table directly without having to manually build the schema and fields
func ObjectToTableWithFormatter(obj interface{}, fieldNameFormatter FieldNameFormatter) (*Table, error) {
	var data []interface{}
	var schema []SchemaField

	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)

	if t.Kind() != reflect.Struct {
		panic(fmt.Errorf("Only struct types are supported. This is %v", t.Kind()))
	}

	for i := 0; i < t.NumField(); i++ {

		fieldName := fieldNameFormatter.Format(t.Field(i).Name)

		typeName := 0

		switch t.Field(i).Type.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			typeName = TypeInt
			data = append(data, int(v.Field(i).Int()))
		case reflect.String:
			typeName = TypeString
			data = append(data, v.Field(i).String())
		case reflect.Float32, reflect.Float64:
			typeName = TypeFloat
			data = append(data, v.Field(i).Float())
		default:
			typeName = TypeString
			s, err := yaml.Marshal(v.Field(i).Interface())
			if err != nil {
				return nil, err
			}
			data = append(data, strings.TrimSpace(string(s)))
		}

		schema = append(schema, SchemaField{
			FieldName: fieldName,
			FieldType: typeName,
		})
	}
	newData := [][]interface{}{data}
	newTbl := Table{newData, schema}
	return &newTbl, nil

}

//RenderRawObject renders an object without having to build a schema for it
func RenderRawObject(obj interface{}, format string, prefixToStrip string) (string, error) {

	switch format {
	case "json", "JSON":
		ret, err := json.MarshalIndent(obj, "", "\t")
		if err != nil {
			return "", err
		}
		return string(ret), nil
	case "csv", "CSV":
		t, err := ObjectToTableWithFormatter(obj, NewPassThroughFormatter())
		if err != nil {
			return "", err
		}
		ret, err := getTableAsCSVString(t.Data, t.Schema)
		if err != nil {
			return "", err
		}
		return ret, nil
	case "yaml", "YAML":
		ret, err := yaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(ret), nil
	case "":
		table, err := ObjectToTableWithFormatter(obj, NewStripPrefixFormatter(prefixToStrip))
		if err != nil {
			return "", err
		}
		ret, err := table.RenderTransposedTableHumanReadable("", "")
		if err != nil {
			return "", err
		}
		return ret, nil
	default:
		return "", fmt.Errorf("Invalid format '%s' given. Valid values are json, JSON, csv, CSV, yaml or YAML.", format)
	}

}
