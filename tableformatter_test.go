package tableformatter

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"encoding/csv"

	"strings"

	"gopkg.in/yaml.v2"

	metalcloud "github.com/bigstepinc/metal-cloud-sdk-go"
	. "github.com/onsi/gomega"
)

func TestTableSortWithSchema(t *testing.T) {

	data := [][]interface{}{
		{4, "str", 20.1},
		{6, "st11r", 22.1},
		{5, "wt11r444", 2.3},
		{5, "wt11r444", 2.1},
		{5, "at11r43", 2.2},
		{4, "xxxx", 2.2},
	}

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}

	TableSorter(schema).OrderBy("LABEL", "ID", "INST.").Sort(data)

	expected := [][]interface{}{
		{5, "at11r43", 2.2},
		{6, "st11r", 22.1},
		{4, "str", 20.1},
		{5, "wt11r444", 2.1},
		{5, "wt11r444", 2.3},
		{4, "xxxx", 2.2},
	}

	if !reflect.DeepEqual(data, expected) {
		t.Errorf("the sorted array was not correct \nwas:\n%+v\n expected\n %+v\n", data, expected)
	}
}

func TestTableSortWithSchemaWithDateTime(t *testing.T) {

	data := [][]interface{}{
		{4, "str", "2013-11-29T13:00:01Z"},
		{6, "st11r", "2013-11-29T13:00:02Z"},
		{6, "st11r", "2014-11-29T13:00:03Z"},
		{6, "st11r", "2012-11-29T13:00:03Z"},
	}

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:   "DATE",
			FieldType:   TypeDateTime,
			FieldSize:   6,
			FieldFormat: defaultTimeFormat,
		},
	}

	TableSorter(schema).OrderBy("DATE").Sort(data)

	expected := [][]interface{}{
		{6, "st11r", "2012-11-29T13:00:03Z"},
		{4, "str", "2013-11-29T13:00:01Z"},
		{6, "st11r", "2013-11-29T13:00:02Z"},
		{6, "st11r", "2014-11-29T13:00:03Z"},
	}

	if !reflect.DeepEqual(data, expected) {
		t.Errorf("the sorted array was not correct \nwas:\n%+v\n expected\n %+v\n", data, expected)
	}
}

func TestDefaultTimeFormat(t *testing.T) {

	layout := defaultTimeFormat

	s := "2012-11-29T13:00:03Z"

	tm, err := time.Parse(layout, s)

	if err != nil {
		t.Errorf("error converting time string %s", err)
	}

	if tm.Year() != 2012 || tm.Second() != 3 {
		t.Error("Date was not parsed correctly")
	}

}

func TestGetTableAsJSONRegressionTest1(t *testing.T) {
	RegisterTestingT(t)
	fw1 := metalcloud.FirewallRule{
		FirewallRuleDescription:    "test desc",
		FirewallRuleProtocol:       "tcp",
		FirewallRulePortRangeStart: 22,
		FirewallRulePortRangeEnd:   23,
	}

	fw2 := metalcloud.FirewallRule{
		FirewallRuleProtocol:       "udp",
		FirewallRulePortRangeStart: 22,
		FirewallRulePortRangeEnd:   22,
	}

	fw3 := metalcloud.FirewallRule{
		FirewallRuleProtocol:                  "tcp",
		FirewallRulePortRangeStart:            22,
		FirewallRulePortRangeEnd:              22,
		FirewallRuleSourceIPAddressRangeStart: "192.168.0.1",
		FirewallRuleSourceIPAddressRangeEnd:   "192.168.0.1",
	}

	fw4 := metalcloud.FirewallRule{
		FirewallRuleProtocol:                  "tcp",
		FirewallRulePortRangeStart:            22,
		FirewallRulePortRangeEnd:              22,
		FirewallRuleSourceIPAddressRangeStart: "192.168.0.1",
		FirewallRuleSourceIPAddressRangeEnd:   "192.168.0.100",
	}

	iao := metalcloud.InstanceArrayOperation{
		InstanceArrayID:           11,
		InstanceArrayLabel:        "testia-edited",
		InstanceArrayDeployType:   "edit",
		InstanceArrayDeployStatus: "not_started",
		InstanceArrayFirewallRules: []metalcloud.FirewallRule{
			fw1,
			fw2,
			fw3,
			fw4,
		},
	}

	ia := metalcloud.InstanceArray{
		InstanceArrayID:            11,
		InstanceArrayLabel:         "testia",
		InfrastructureID:           100,
		InstanceArrayOperation:     &iao,
		InstanceArrayServiceStatus: "active",
		InstanceArrayFirewallRules: []metalcloud.FirewallRule{
			fw1,
			fw2,
			fw3,
			fw4,
		},
	}

	list := ia.InstanceArrayOperation.InstanceArrayFirewallRules
	data := [][]interface{}{}
	idx := 0

	for _, fw := range list {

		portRange := "any"

		if fw.FirewallRulePortRangeStart != 0 {
			portRange = fmt.Sprintf("%d", fw.FirewallRulePortRangeStart)
		}

		if fw.FirewallRulePortRangeStart != fw.FirewallRulePortRangeEnd {
			portRange += fmt.Sprintf("-%d", fw.FirewallRulePortRangeEnd)
		}

		sourceIPRange := "any"

		if fw.FirewallRuleSourceIPAddressRangeStart != "" {
			sourceIPRange = fw.FirewallRuleSourceIPAddressRangeStart
		}

		if fw.FirewallRuleSourceIPAddressRangeStart != fw.FirewallRuleSourceIPAddressRangeEnd {
			sourceIPRange += fmt.Sprintf("-%s", fw.FirewallRuleSourceIPAddressRangeEnd)
		}

		data = append(data, []interface{}{
			idx,
			fw.FirewallRuleProtocol,
			portRange,
			sourceIPRange,
			fw.FirewallRuleEnabled,
			fw.FirewallRuleDescription,
		})

		idx++

	}

	schema := []SchemaField{
		{
			FieldName: "INDEX",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "PROTOCOL",
			FieldType: TypeString,
			FieldSize: 10,
		},
		{
			FieldName: "PORT",
			FieldType: TypeString,
			FieldSize: 10,
		},
		{
			FieldName: "SOURCE",
			FieldType: TypeString,
			FieldSize: 20,
		},

		{
			FieldName: "ENABLED",
			FieldType: TypeBool,
			FieldSize: 10,
		},
		{
			FieldName: "DESC.",
			FieldType: TypeString,
			FieldSize: 50,
		},
	}

	Expect(data[0][0]).NotTo(Equal(data[0][1]))
	Expect(data[0][1]).NotTo(Equal(data[0][2]))
	Expect(data[0][1]).NotTo(Equal(data[0][2]))

	ret, err := getTableAsJSONString(data, schema)
	Expect(err).To(BeNil())

	var m []interface{}
	err = json.Unmarshal([]byte(ret), &m)
	Expect(err).To(BeNil())

	Expect(m[0].(map[string]interface{})["INDEX"]).ToNot(Equal(m[1].(map[string]interface{})["INDEX"]))
	Expect(m[0].(map[string]interface{})["INDEX"]).ToNot(Equal(m[2].(map[string]interface{})["INDEX"]))
	Expect(m[1].(map[string]interface{})["INDEX"]).ToNot(Equal(m[2].(map[string]interface{})["INDEX"]))
}

func TestGetTableHeader(t *testing.T) {

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}
	expected := "| ID    | LABEL               | INST. |"

	actual := getTableHeader(schema)

	if actual != expected {
		t.Errorf("Header is not correct, \nexpected:  %s\n     was: %s", expected, actual)
	}
}

func TestGetTableRow(t *testing.T) {
	RegisterTestingT(t)
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
		{
			FieldName: "INTF",
			FieldSize: 5,
			FieldType: TypeInterface,
		},
	}

	row := []interface{}{10, "test", 33.3, map[string]string{"test": "test1", "test2": "test3"}}

	actual := getTableRow(row, schema)

	Expect(actual).To(ContainSubstring("test1"))
	Expect(actual).To(ContainSubstring("test3"))
}

func TestGetTableDelimiter(t *testing.T) {
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}

	expected := "+-------+---------------------+-------+"

	actual := getTableDelimiter(schema)

	if actual != expected {
		t.Errorf("Delimiter is not correct, \nexpected: %s\n     was: %s", expected, actual)
	}
}

func TestGetTableAsString(t *testing.T) {
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}

	expected :=
		`+-------+---------------------+-------+
| ID    | LABEL               | INST. |
+-------+---------------------+-------+
| 4     | str                 | 20.10 |
| 5     | st11r               | 22.10 |
| 6     | st11r444            | 2.10  |
+-------+---------------------+-------+
`
	data := [][]interface{}{
		{4, "str", 20.1},
		{5, "st11r", 22.1},
		{6, "st11r444", 2.1},
	}

	actual := getTableAsString(data, schema)

	if actual != expected {
		t.Errorf("Delimiter is not correct, \nexpected:\n%s\nwas:\n%s", expected, actual)
	}
}

func TestGetTableAsJSONString(t *testing.T) {
	RegisterTestingT(t)
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}

	data := [][]interface{}{
		{4, "str", 20.1},
		{5, "st11r", 22.1},
		{6, "st11r444", 2.1},
	}

	ret, err := getTableAsJSONString(data, schema)
	if err != nil {
		t.Errorf("%s", err)
	}

	var m []interface{}

	err = json.Unmarshal([]byte(ret), &m)
	Expect(err).To(BeNil())

	Expect(int(m[0].(map[string]interface{})["ID"].(float64))).To(Equal(data[0][0]))
	Expect(m[0].(map[string]interface{})["LABEL"]).To(Equal(data[0][1]))
	Expect(m[2].(map[string]interface{})["LABEL"]).To(Equal(data[2][1]))
	Expect(float64(m[2].(map[string]interface{})["INST."].(float64))).To(Equal(data[2][2]))

}

func TestGetTableAsCSVString(t *testing.T) {

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}

	expected :=
		`ID,LABEL,INST.
4,str,20.100000
5,st11r,22.100000
6,st11r444,2.100000
`

	data := [][]interface{}{
		{4, "str", 20.1},
		{5, "st11r", 22.1},
		{6, "st11r444", 2.1},
	}

	actual, err := getTableAsCSVString(data, schema)
	if err != nil {
		t.Errorf("%s", err)
	}
	if actual != expected {
		t.Errorf("Delimiter is not correct, \nexpected:\n%s\nwas:\n%s", expected, actual)
	}
}

func TestAdjustFieldSizes(t *testing.T) {
	RegisterTestingT(t)
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 3,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 5, //this is smaller than the largest
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      0,
			FieldPrecision: 4,
		},
		{
			FieldName:      "VERY LONG FIELD NAME",
			FieldType:      TypeString,
			FieldSize:      4,
			FieldPrecision: 4,
		},
	}

	data := [][]interface{}{
		{4, "12345", 20.1, "tes"},
		{5, "12", 22.1, "te"},
		{6, "12345\n6789", 1.2345, "t"},
	}
	table := Table{data, schema}
	table.AdjustFieldSizes()

	Expect(schema[0].FieldSize).To(Equal(3))
	Expect(schema[1].FieldSize).To(Equal(5))
	Expect(schema[2].FieldSize).To(Equal(8))
	//test if expands with LABEl
	Expect(schema[3].FieldSize).To(Equal(21))

}

func TestRenderTable(t *testing.T) {
	RegisterTestingT(t)
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 3,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 5,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      0,
			FieldPrecision: 4,
		},
		{
			FieldName:      "VERY LONG FIELD NAME",
			FieldType:      TypeString,
			FieldSize:      4,
			FieldPrecision: 4,
		},
	}

	data := [][]interface{}{
		{4, "12345", 20.1, "tes"},
		{5, "12", 22.1, "te"},
		{6, "123456789", 1.2345, "t"},
	}

	table := Table{data, schema}
	s, err := table.RenderTable("test", "", "")

	Expect(err).To(BeNil())
	Expect(s).To(ContainSubstring("test"))
	Expect(s).To(ContainSubstring("VERY LONG"))

	s, err = table.RenderTable("test", "", "json")
	Expect(err).To(BeNil())
	var m []interface{}
	err = json.Unmarshal([]byte(s), &m)
	Expect(err).To(BeNil())

	s, err = table.RenderTable("test", "", "csv")
	Expect(err).To(BeNil())

	s, err = table.RenderTable("test", "", "yaml")
	err = yaml.Unmarshal([]byte(s), &m)
	Expect(err).To(BeNil())
}

func TestYAMLMArshalOfMetalcloudObjects(t *testing.T) {
	RegisterTestingT(t)

	var sw metalcloud.SwitchDevice

	err := json.Unmarshal([]byte(_switchDeviceFixture1), &sw)
	Expect(err).To(BeNil())

	b, err := yaml.Marshal(sw)
	Expect(err).To(BeNil())

	t.Log(string(b))

	var sw2 metalcloud.SwitchDevice

	err = yaml.Unmarshal(b, &sw2)
	Expect(err).To(BeNil())
	Expect(sw2.NetworkEquipmentPrimaryWANIPv4SubnetPool).To(Equal(sw.NetworkEquipmentPrimaryWANIPv4SubnetPool))
	//for some reason this doesn't work. don't know why yet
	//t.Logf("sw1=%+v", sw)
	//t.Logf("sw2=%+v", sw2)
	//Expect(reflect.DeepEqual(sw, sw2)).To(BeTrue())
}

func TestYAMLMArshalCaseSensitivity(t *testing.T) {
	RegisterTestingT(t)

	type dummy struct {
		WithCamelCase1 string
		WithCamelCase2 int `yaml:"withCamelCase2"`
		A              int
	}

	var d dummy

	s := `
withcamelcase1: test
withCamelCase2: 10
a: 12
`

	err := yaml.Unmarshal([]byte(s), &d)
	Expect(err).To(BeNil())
	Expect(d.A).To(Equal(12))
	Expect(d.WithCamelCase2).To(Equal(10))
	Expect(d.WithCamelCase1).To(Equal("test"))

}

func JSONUnmarshal(jsonString string) ([]interface{}, error) {
	var m []interface{}
	err := json.Unmarshal([]byte(jsonString), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

//JSONFirstRowEquals checks if values of the table returned in the json match the values provided. Type is not checked (we check string equality)
func JSONFirstRowEquals(jsonString string, testVals map[string]interface{}) error {
	m, err := JSONUnmarshal(jsonString)
	if err != nil {
		return err
	}

	firstRow := m[0].(map[string]interface{})

	for k, v := range testVals {
		if fmt.Sprintf("%+v", firstRow[k]) != fmt.Sprintf("%+v", v) {
			return fmt.Errorf("values for key %s do not match:  expected '%+v' provided '%+v'", k, v, firstRow[k])
		}
	}

	return nil
}

func CSVUnmarshal(csvString string) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(csvString))

	return reader.ReadAll()
}

//CSVFirstRowEquals checks if values of the table returned in the json match the values provided. Type is not checked (we check string equality)
func CSVFirstRowEquals(csvString string, testVals map[string]interface{}) error {
	m, err := CSVUnmarshal(csvString)
	if err != nil {
		return err
	}

	header := m[0]
	firstRow := map[string]string{}
	//turn first row into a map
	for k, v := range m[1] {
		firstRow[header[k]] = v
	}

	for k, v := range testVals {
		if fmt.Sprintf("%+v", firstRow[k]) != fmt.Sprintf("%+v", v) {
			return fmt.Errorf("values for key %s do not match:  expected '%+v' provided '%+v'", k, v, firstRow[k])
		}
	}

	return nil
}

func TestTransposeTable(t *testing.T) {
	RegisterTestingT(t)
	data := [][]interface{}{
		{11, 12, 13},
		{21, 22, 23},
		{31, 32, 33},
	}
	table := Table{data, nil}
	tableT := TransposeTable(table)

	expectedDataT := [][]interface{}{
		{11, 21, 31},
		{12, 22, 32},
		{13, 23, 33},
	}

	Expect(tableT.Data).Should(Equal(expectedDataT))
}

func TestConvertToStringTable(t *testing.T) {
	RegisterTestingT(t)
	data := [][]interface{}{
		{11, "12", 13.4},
		{21, "22", 23.3},
		{31, "32", 33.4},
	}
	table := Table{
		Data:   data,
		Schema: []SchemaField{},
	}
	stringsTable := ConvertToStringTable(table)

	expectedDataT := [][]interface{}{
		{"11", "12", "13.4"},
		{"21", "22", "23.3"},
		{"31", "32", "33.4"},
	}

	Expect(stringsTable.Data).Should(Equal(expectedDataT))
}

func TestRenderTransposedTable(t *testing.T) {
	RegisterTestingT(t)
	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 3,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 5,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      0,
			FieldPrecision: 4,
		},
		{
			FieldName:      "VERY LONG FIELD NAME",
			FieldType:      TypeString,
			FieldSize:      4,
			FieldPrecision: 4,
		},
	}

	data := [][]interface{}{
		{4, "12345", 20.1, "tes"},
	}

	table := Table{data, schema}
	s, err := table.RenderTransposedTable("test", "", "")

	Expect(err).To(BeNil())
	Expect(s).To(ContainSubstring("KEY"))
	Expect(s).To(ContainSubstring("VALUE"))
	Expect(s).To(ContainSubstring("12345"))
	Expect(s).To(ContainSubstring("20.1"))

	s, err = table.RenderTransposedTable("test", "", "json")
	Expect(err).To(BeNil())
	var m []interface{}
	err = json.Unmarshal([]byte(s), &m)
	Expect(err).To(BeNil())

	s, err = table.RenderTransposedTable("test", "", "csv")
	Expect(err).To(BeNil())
}

func TestObjectToTable(t *testing.T) {

	RegisterTestingT(t)

	var sw metalcloud.SwitchDevice

	err := json.Unmarshal([]byte(_switchDeviceFixture1), &sw)
	Expect(err).To(BeNil())

	table, err := ObjectToTable(sw)

	Expect(err).To(BeNil())
	Expect(len(table.Data[0])).To(Equal(40))
	Expect(table.Data[0][1]).To(Equal("UK_RDG_EVR01_00_0001_00A9_01"))
	Expect(table.Schema[1].FieldName).To(Equal("network equipment identifier string"))
	Expect(table.Schema[39].FieldName).To(Equal("volume template id"))
	Expect(table.Data[0][39]).To(Equal(0))
}

func TestObjToTableWithFormatter(t *testing.T) {
	RegisterTestingT(t)

	var sw metalcloud.SwitchDevice

	err := json.Unmarshal([]byte(_switchDeviceFixture1), &sw)
	Expect(err).To(BeNil())

	table, err := ObjectToTableWithFormatter(sw, NewStripPrefixFormatter("NetworkEquipment"))
	Expect(err).To(BeNil())
	Expect(len(table.Data[0])).To(Equal(40))
	Expect(table.Data[0][1]).To(Equal("UK_RDG_EVR01_00_0001_00A9_01"))
	Expect(table.Schema[1].FieldName).To(Equal("Identifier String"))
	Expect(table.Schema[39].FieldName).To(Equal("Volume Template Id"))
	Expect(table.Data[0][39]).To(Equal(0))

}

func TestRenderTransposedTableHumanReadable(t *testing.T) {
	RegisterTestingT(t)

	schema := []SchemaField{
		{
			FieldName: "Field1",
			FieldType: TypeInt,
		},
		{
			FieldName: "Field2",
			FieldType: TypeString,
		},
	}

	data := [][]interface{}{
		{
			10,
			"test",
		},
	}

	table := Table{data, schema}
	s, err := table.RenderTransposedTableHumanReadable("test", "test")

	Expect(err).To(BeNil())
	Expect(s).To(Equal(`Field1: 10
Field2: test
`))

}

func TestRenderRawObject(t *testing.T) {
	RegisterTestingT(t)

	var sw metalcloud.SwitchDevice

	err := json.Unmarshal([]byte(_switchDeviceFixture1), &sw)
	Expect(err).To(BeNil())

	ret, err := RenderRawObject(sw, "json", "")

	Expect(err).To(BeNil())
	Expect(ret).NotTo(BeNil())
	Expect(ret).To(ContainSubstring("2A02:0CB8:0000:0000:0000:0000:0000:0000/53"))

	var sw2 metalcloud.SwitchDevice
	err = json.Unmarshal([]byte(ret), &sw2)
	Expect(err).To(BeNil())

	ret, err = RenderRawObject(sw, "yaml", "")
	Expect(err).To(BeNil())
	Expect(ret).NotTo(BeNil())
	Expect(ret).To(ContainSubstring("2A02:0CB8:0000:0000:0000:0000:0000:0000/53"))

	ret, err = RenderRawObject(sw, "csv", "")
	Expect(err).To(BeNil())
	Expect(ret).NotTo(BeNil())
	Expect(ret).To(ContainSubstring("2A02:0CB8:0000:0000:0000:0000:0000:0000/53"))

	ret, err = RenderRawObject(sw, "", "NetworkEquipment")
	Expect(err).To(BeNil())
	Expect(ret).NotTo(BeNil())
	Expect(ret).To(ContainSubstring("2A02:0CB8:0000:0000:0000:0000:0000:0000/53"))

	t.Log(ret)

}

func TestGetRowSize(t *testing.T) {
	RegisterTestingT(t)

	data := [][]interface{}{
		{4, "12345", 20.1, "tes"},
		{5, "12", 22.1, "te"},
		{6, "123456789", 1.2345, "t"},
	}

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 3,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 5,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      0,
			FieldPrecision: 4,
		},
		{
			FieldName:      "VERY LONG FIELD NAME",
			FieldType:      TypeString,
			FieldSize:      4,
			FieldPrecision: 4,
		},
	}

	s := getRowSize(data, schema)
	Expect(s).To(Equal(16))
}

func TestGetTableRowMultiline(t *testing.T) {
	RegisterTestingT(t)

	data := [][]interface{}{
		{4, "123\n45", 20.1, "teklkkkllklklsas\ndasda\nsdasd"},
		{5, "12", 22.1, "te"},
		{6, "123456789", 1.2345, "t"},
	}

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 3,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 5,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      0,
			FieldPrecision: 4,
		},
		{
			FieldName:      "VERY LONG FIELD NAME",
			FieldType:      TypeString,
			FieldSize:      4,
			FieldPrecision: 4,
		},
	}

	expected :=
		`| 4  | 123  | 20.1000| teklkkkllklklsas|
|    | 45   |        | dasda           |
|    |      |        | sdasd           |`

	s := getTableRow(data[0], schema)
	t.Logf("%s", s)

	Expect(s).To(Equal(expected))
}

func TestGetFoldedTableAsString(t *testing.T) {
	RegisterTestingT(t)

	data := [][]interface{}{
		{4, "12345", 20.1, "tes"},
		{5, "12", 22.1, "te"},
		{6, "123456789", 1.2345, "t"},
	}

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 3,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 5,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      0,
			FieldPrecision: 4,
		},
		{
			FieldName:      "VERY LONG FIELD NAME",
			FieldType:      TypeString,
			FieldSize:      4,
			FieldPrecision: 4,
		},
	}

	expected :=
		`+--------------------------+
| Values                   |
+--------------------------+
| - id: 4                  |
|   inst: 20.1             |
|   label: "12345"         |
|   veryLongFieldName: tes |
|                          |
| - id: 5                  |
|   inst: 22.1             |
|   label: "12"            |
|   veryLongFieldName: te  |
|                          |
| - id: 6                  |
|   inst: 1.2345           |
|   label: "123456789"     |
|   veryLongFieldName: t   |
|                          |
+--------------------------+
`

	s, err := getFoldedTableAsString(data, schema)
	t.Logf("%s", s)
	Expect(err).To(BeNil())
	Expect(s).To(Equal(expected))
}

func colorize(str string, color string) string {
	colorReset := "\033[0m"
	var c string
	switch color {
	case "red":
		c = "\033[12;41m"
	case "green":
		c = "\033[33m"
	default:
		c = colorReset
	}

	return fmt.Sprintf("%s%s%s", c, str, colorReset)

}

func TestRenderTableWithColor(t *testing.T) {
	RegisterTestingT(t)
	data := [][]interface{}{
		{4, colorize("str", "red"), 20.1},
		{6, "st11r", 22.1},
		{5, "wt11r444", 2.3},
		{5, "wt11r444", 2.1},
		{5, colorize("at11r43", "green"), 2.2},
		{4, "xxxx", 2.2},
	}

	schema := []SchemaField{
		{
			FieldName: "ID",
			FieldType: TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: TypeString,
			FieldSize: 20,
		},
		{
			FieldName:      "INST.",
			FieldType:      TypeFloat,
			FieldSize:      6,
			FieldPrecision: 2,
		},
	}

	table := Table{data, schema}
	s, err := table.RenderTable("test", "", "")

	Expect(err).To(BeNil())
	Expect(s).To(ContainSubstring("| LABEL               | "))

}

func TestDeColorize(t *testing.T) {
	RegisterTestingT(t)

	field := SchemaField{
		FieldName: "INST.",
		FieldType: TypeString,
		FieldSize: 1,
	}

	o := "  test   "
	s := colorize(o, "red")
	d := decolorize(s)

	Expect(getCellSize(s, &field)).To(Equal(getCellSize(o, &field)))

	Expect(d).To(Equal(o))

}

func TestPad(t *testing.T) {
	RegisterTestingT(t)

	o := "test"
	s := pad(o, 7)

	Expect(s).To(Equal("test   "))

}

const _switchDeviceFixture1 = "{\"network_equipment_id\":1,\"datacenter_name\":\"uk-reading\",\"network_equipment_driver\":\"hp5900\",\"network_equipment_position\":\"tor\",\"network_equipment_provisioner_type\":\"vpls\",\"network_equipment_identifier_string\":\"UK_RDG_EVR01_00_0001_00A9_01\",\"network_equipment_description\":\"HP Comware Software, Version 7.1.045, Release 2311P06\",\"network_equipment_management_address\":\"10.0.0.0\",\"network_equipment_management_port\":22,\"network_equipment_management_username\":\"sad\",\"network_equipment_quarantine_vlan\":5,\"network_equipment_quarantine_subnet_start\":\"11.16.0.1\",\"network_equipment_quarantine_subnet_end\":\"11.16.0.00\",\"network_equipment_quarantine_subnet_prefix_size\":24,\"network_equipment_quarantine_subnet_gateway\":\"11.16.0.1\",\"network_equipment_primary_wan_ipv4_subnet_pool\":\"11.24.0.2\",\"network_equipment_primary_wan_ipv4_subnet_prefix_size\":22,\"network_equipment_primary_san_subnet_pool\":\"100.64.0.0\",\"network_equipment_primary_san_subnet_prefix_size\":21,\"network_equipment_primary_wan_ipv6_subnet_pool_id\":1,\"network_equipment_primary_wan_ipv6_subnet_cidr\":\"2A02:0CB8:0000:0000:0000:0000:0000:0000/53\",\"network_equipment_cached_updated_timestamp\":\"2020-08-04T20:11:49Z\",\"network_equipment_management_protocol\":\"ssh\",\"chassis_rack_id\":null,\"network_equipment_cache_wrapper_json\":null,\"network_equipment_cache_wrapper_phpserialize\":\"\",\"network_equipment_tor_linked_id\":null,\"network_equipment_uplink_ip_addresses_json\":null,\"network_equipment_management_address_mask\":null,\"network_equipment_management_address_gateway\":null,\"network_equipment_requires_os_install\":false,\"network_equipment_management_mac_address\":\"00:00:00:00:00:00\",\"volume_template_id\":null,\"network_equipment_country\":null,\"network_equipment_city\":null,\"network_equipment_datacenter\":null,\"network_equipment_datacenter_room\":null,\"network_equipment_datacenter_rack\":null,\"network_equipment_rack_position_upper_unit\":null,\"network_equipment_rack_position_lower_unit\":null,\"network_equipment_serial_numbers\":null,\"network_equipment_info_json\":null,\"network_equipment_management_subnet\":null,\"network_equipment_management_subnet_prefix_size\":null,\"network_equipment_management_subnet_start\":null,\"network_equipment_management_subnet_end\":null,\"network_equipment_management_subnet_gateway\":null,\"datacenter_id_parent\":null,\"network_equipment_dhcp_packet_sniffing_is_enabled\":1,\"network_equipment_driver_dump_cached_json\":null,\"network_equipment_tags\":[],\"network_equipment_management_password\":\"ddddd\"}"
