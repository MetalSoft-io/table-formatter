# golang table formatter

[https://travis-ci.org/MetalSoft-io/tableformatter.svg?branch=main]
[![GoDoc](https://godoc.org/github.com/metalsoft-io/tableformatter?status.svg)](https://godoc.org/github.com/metalsoft-io/tableformatter)
[![Build Status](https://travis-ci.org/MetalSoft-io/tableformatter.svg?branch=main)](https://travis-ci.org/MetalSoft-io/tableformatter)

Utility to print pretty text (ASCII) tables. It supports:

* automatic field size adjustment
* yaml, json, CSV alternative rendering
* multi-line cells

```
Employee list:
+-------+-----------------------------------------------------+---------------------+-----------+--------+--------------------------------+
| ID    | LABEL                                               | OWNER               | REL.      | STATUS | DATACENTER                     |
+-------+-----------------------------------------------------+---------------------+-----------+--------+--------------------------------+
| 10    | test-infrastructure                                 | alex@alex.com       | manager   | active | uk-reading                     |
| 20    | production-infrastructure                           | john@alex.com       | CTO       | active | us-santaclara                  |
| 34    | production-infrastructure                           | john@alex.com       | CTO       | active | us-santaclara                  |
|       | another line                                        |                     |           |        | multiline-string               |
|       | another line                                        |                     |           |        |                                |
+-------+-----------------------------------------------------+---------------------+-----------+--------+--------------------------------+
Total: 3 employees
```

Folded version:

```
Employee list:
+------------------------------------+
| Values                             |
+------------------------------------+
| - datacenter: uk-reading           |
|   id: 10                           |
|   label: test-infrastructure       |
|   owner: alex@alex.com             |
|   rel: manager                     |
|   status: active                   |
|                                    |
| - datacenter: us-santaclara        |
|   id: 20                           |
|   label: production-infrastructure |
|   owner: john@alex.com             |
|   rel: CTO                         |
|   status: active                   |
|                                    |
| - datacenter: |-                   |
|     us-santaclara                  |
|     multiline-string               |
|   id: 34                           |
|   label: |-                        |
|     production-infrastructure      |
|     another line                   |
|     another line                   |
|   owner: john@alex.com             |
|   rel: CTO                         |
|   status: active                   |
|                                    |
+------------------------------------+
Total: 3 employees
```

By default it will automatically fold at 100 chars.

## Example

```golang
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
```