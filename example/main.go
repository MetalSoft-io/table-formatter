package main

import (
	"fmt"

	"github.com/metalsoft/tableformatter"
)

func main() {

	schema := []tableformatter.SchemaField{
		{
			FieldName: "ID",
			FieldType: tableformatter.TypeInt,
			FieldSize: 6,
		},
		{
			FieldName: "LABEL",
			FieldType: tableformatter.TypeString,
			FieldSize: 15,
		},
		{
			FieldName: "OWNER",
			FieldType: tableformatter.TypeString,
			FieldSize: 20,
		},
		{
			FieldName: "REL.",
			FieldType: tableformatter.TypeString,
			FieldSize: 10,
		},
		{
			FieldName: "STATUS",
			FieldType: tableformatter.TypeString,
			FieldSize: 5,
		},
		{
			FieldName: "DATACENTER",
			FieldType: tableformatter.TypeString,
			FieldSize: 10,
		},
	}

	data := [][]interface{}{
		{
			10,
			"test-infrastructure",
			"alex@alex.com",
			"manager",
			"active",
			"uk-reading",
		},
		{
			20,
			"production-infrastructure",
			"john@alex.com",
			"CTO",
			"active",
			"us-santaclara",
		},
		{
			34,
			"production-infrastructure\nanother line\nanother line",
			"john@alex.com",
			"CTO",
			"active",
			"us-santaclara\nmultiline-string",
		},
	}

	table := tableformatter.Table{
		Data:   data,
		Schema: schema,
	}

	s, err := table.RenderTable("employees", "Employee list:", "text")
	if err != nil {
		fmt.Printf("%+v", err)
	}
	fmt.Printf("%s", s)

}
