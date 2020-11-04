package tableformatter

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/iancoleman/strcase"
	"gopkg.in/yaml.v2"
)

//Table represents a table including data and schema
type Table struct {
	data   [][]interface{}
	schema []SchemaField
}

const defaultTimeFormat = "2006-01-02T15:04:05Z" //oddly enough this is how you specify a format

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

				layout := defaultTimeFormat

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

//ConsoleIOChannel represents an IO channel, typically stdin and stdout but could be anything
type ConsoleIOChannel struct {
	Stdin  io.Reader
	Stdout io.Writer
}

var consoleIOChannelInstance ConsoleIOChannel

var once sync.Once

//GetConsoleIOChannel returns the console channel singleton
func GetConsoleIOChannel() *ConsoleIOChannel {
	once.Do(func() {

		consoleIOChannelInstance = ConsoleIOChannel{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
		}
	})

	return &consoleIOChannelInstance
}

//GetStdout returns the configured output channel
func GetStdout() io.Writer {
	return GetConsoleIOChannel().Stdout
}

//GetStdin returns the configured input channel
func GetStdin() io.Reader {
	return GetConsoleIOChannel().Stdin
}

//SetConsoleIOChannel configures the stdin and stdout to be used by all io with
func SetConsoleIOChannel(in io.Reader, out io.Writer) {
	channel := GetConsoleIOChannel()
	channel.Stdin = in
	channel.Stdout = out
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

//getTableRow returns the string for a row with the | delimiter
func getTableRow(row []interface{}, schema []SchemaField) string {
	var rowStr []string
	for i, field := range schema {
		switch field.FieldType {
		case TypeInt:
			rowStr = append(rowStr, fmt.Sprintf(fmt.Sprintf(" %%-%dd", field.FieldSize), row[i].(int)))
		case TypeString:
			//escape %
			s := strings.ReplaceAll(row[i].(string), "%", "%%")
			rowStr = append(rowStr, fmt.Sprintf(fmt.Sprintf(" %%-%ds", field.FieldSize), s))
		case TypeFloat:
			rowStr = append(rowStr, fmt.Sprintf(fmt.Sprintf(" %%-%d.%df", field.FieldSize, field.FieldPrecision), row[i].(float64)))
		default:
			rowStr = append(rowStr, fmt.Sprintf(fmt.Sprintf(" %%-+%dv", field.FieldSize), row[i]))
		}
	}
	return "|" + strings.Join(rowStr, "|") + "|"
}

// GetCellSize calculates how wide a cell is by converting it to string and measuring it's size
func getCellSize(d interface{}, field *SchemaField) int {
	var s string
	switch field.FieldType {
	case TypeInt:
		s = fmt.Sprintf("%d", d.(int))
	case TypeString:
		s = d.(string)
	case TypeFloat:
		s = fmt.Sprintf(fmt.Sprintf("%%.%df", field.FieldPrecision), d.(float64))
	default:
		s = fmt.Sprintf("%v", d)

	}
	return len(s)
}

//getRowSize returns the row size of a table
func getRowSize(data [][]interface{}, schema *[]SchemaField) int {
	rowSize := len(*schema)
	size := 0
	for i := 0; i < rowSize; i++ {
		f := (*schema)[i]
		size += getCellSize(data[0][i], &f)
	}
	return size
}

//AdjustFieldSizes expands field sizes to match the widest cell
func (t *Table) AdjustFieldSizes() {

	rowSize := len(t.schema)
	for i := 0; i < rowSize; i++ {
		f := t.schema[i]

		//iterate over the entire column
		rowCount := len(t.data)

		maxLen := f.FieldSize

		if len(f.FieldName) > maxLen {
			maxLen = len(f.FieldName)
		}

		for k := 0; k < rowCount; k++ {
			cellSize := getCellSize(t.data[k][i], &f)
			if cellSize > maxLen {
				maxLen = cellSize
			}
		}
		if maxLen > f.FieldSize {
			t.schema[i].FieldSize = maxLen + 1 //we leave a little room to the right
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

	return string(ret), nil
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

	return string(ret), nil
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

	return buf.String(), nil
}

func truncateString(s string, length int) string {
	str := s
	if len(str) > 0 {
		return str[:length] + "..."
	}
	return ""
}

//RenderTable renders a table object as a string
//supported formats: json, csv, yaml
func (t *Table) RenderTable(tableName string, topLine string, format string) (string, error) {
	var sb strings.Builder

	switch format {
	case "json", "JSON":
		ret, err := getTableAsJSONString(t.data, t.schema)
		if err != nil {
			return "", err
		}
		sb.WriteString(ret)
	case "csv", "CSV":
		ret, err := getTableAsCSVString(t.data, t.schema)
		if err != nil {
			return "", err
		}
		sb.WriteString(ret)
	case "yaml", "YAML":
		ret, err := getTableAsYAMLString(t.data, t.schema)
		if err != nil {
			return "", err
		}
		sb.WriteString(ret)

	default:
		if topLine != "" {
			sb.WriteString(fmt.Sprintf("%s\n", topLine))
		}

		t.AdjustFieldSizes()

		sb.WriteString(getTableAsString(t.data, t.schema))

		sb.WriteString(fmt.Sprintf("Total: %d %s\n\n", len(t.data), tableName))
	}

	return sb.String(), nil
}

func getRowWidth(data []interface{}) {

}

//TransposeTable turns columns into rows. It assumes an uniform length table
func TransposeTable(data [][]interface{}) [][]interface{} {

	dataT := [][]interface{}{}

	if len(data) == 0 {
		return dataT
	}

	tableLength := len(data)
	rowLength := len(data[0])

	for j := 0; j < rowLength; j++ {

		newRow := []interface{}{}

		for i := 0; i < tableLength; i++ {

			newRow = append(newRow, data[i][j])
		}

		dataT = append(dataT, newRow)
	}

	return dataT
}

//ConvertToStringTable converts all cells to string cells
func ConvertToStringTable(data [][]interface{}) [][]interface{} {
	dataS := [][]interface{}{}

	for _, row := range data {
		newRow := []interface{}{}
		for _, v := range row {
			if v == nil {
				v = " "
			}
			newRow = append(newRow, fmt.Sprintf("%v", v))
		}
		dataS = append(dataS, newRow)
	}
	return dataS
}

//RenderTransposedTable renders the text format as a key-value table. json and csv formats remain the same as render table
//supported formats: json, csv, yaml
func (t *Table) RenderTransposedTable(tableName string, topLine string, format string) (string, error) {

	if format != "" {
		return t.RenderTable(tableName, topLine, format)
	}

	headerRow := []interface{}{}
	for _, s := range t.schema {
		headerRow = append(headerRow, s.FieldName)
	}

	dataAsStrings := ConvertToStringTable(t.data)
	newData := [][]interface{}{}
	newData = append(newData, headerRow)
	for _, row := range dataAsStrings {
		newData = append(newData, row)
	}

	dataTransposed := TransposeTable(newData)

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

	newTable := Table{
		dataTransposed,
		newSchema,
	}

	return newTable.RenderTable(tableName, topLine, format)

}

//RenderTransposedTableHumanReadable renders an object in a human readable way
func (t *Table) RenderTransposedTableHumanReadable(tableName string, topLine string) (string, error) {

	headerRow := []interface{}{}
	for _, s := range t.schema {
		headerRow = append(headerRow, s.FieldName)
	}

	var sb strings.Builder
	for i, field := range t.schema {
		sb.WriteString(fmt.Sprintf("%s: %v\n", field.FieldName, t.data[0][i]))
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
		ret, err := getTableAsCSVString(t.data, t.schema)
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
	default:
		table, err := ObjectToTableWithFormatter(obj, NewStripPrefixFormatter(prefixToStrip))
		if err != nil {
			return "", err
		}
		ret, err := table.RenderTransposedTableHumanReadable("", "")
		if err != nil {
			return "", err
		}
		return ret, nil
	}

}
