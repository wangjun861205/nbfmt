package nbfmt

import (
	"fmt"
	"log"
	"testing"
)

type Table struct {
	TableName string
}

const src = `type FieldArg interface {
	columnName() string
	tableName() string
	sqlValue() string
}
{{ for _, tab in DB.Tables }}
	{{ for _, col in tab.Columns }}
		{{ switch col.FieldType }}
			{{ case "int64" }}
				type {{ tab.ModelName }}{{ col.FieldName }} int64
				func New{{ tab.ModelName }}{{ col.FieldName }}(val int64) *{{ tab.ModelName }}{{ col.FieldName }} {
					f := {{ tab.ModelName }}{{ col.FieldName }}(val)
					return &f
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) columnName() string {
					return "{{ col.ColumnName }}"
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) sqlValue() string {
					return fmt.Sprintf("%d", *f)
				}
			{{ case "string" }}
				type {{ tab.ModelName }}{{ col.FieldName }} string
				func New{{ tab.ModelName }}{{ col.FieldName}}(val string) *{{ tab.ModelName }}{{ col.FieldName}} {
					f := {{ tab.ModelName }}{{ col.FieldName }}(val)
					return &f
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) columnName() string {
					return "{{ col.ColumnName }}"
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) sqlValue() string {
					return fmt.Sprintf("%q", *f)
				}
			{{ case "float64" }}
				type {{ table.ModelName }}{{ col.FieldName }} float64
				func New{{ tab.ModelName }}{{ col.FieldName }}(val float64) *{{ tab.ModelName }}{{ col.FieldName }} {
					f := {{ tab.ModelName }}{{ col.FieldName }}(val)
					return &f
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) columnName() string {
					return "{{ col.ColumnName }}"
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) sqlValue() string {
					return fmt.Sprintf("%f", *f)
				}
			{{ case "bool" }}
				type {{ tab.ModelName }}{{ col.FieldName }} bool
				func New{{ tab.ModelName }}{{ col.FieldName }}(val bool) *{{ tab.ModelName }}{{ col.FieldName }} {
					f := {{ tab.ModelName }}{{ col.FieldName }}(val)
					return &f
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) columnName() string {
					return "{{ col.ColumnName }}"
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) sqlValue() string {
					return fmt.Sprintf("%t", *f)
				}
			{{ case "time.Time" }}
				type {{ tab.ModelName }}{{ col.FieldName }} time.Time
				func New{{ tab.ModelName }}{{ col.FieldName }}(val time.Time) *{{ tab.ModelName }}{{ col.FieldName }}{
					f := {{ tab.ModelName }}{{ col.FieldName }}(val)
					return &f
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) columnName() string {
					return "{{ col.ColumnName }}"
				}
				func (f *{{ tab.ModelName }}{{ col.FieldName }}) sqlValue() string {
					{{ switch col.MySqlType }}
						{{ case "DATE" }}
							return time.Time(*f).Format("2006-01-02")
						{{ case "DATETIME" }}
							return time.Time(*f).Format("2006-01-02 15:04:05")
						{{ case "TIMESTAMP" }}
							return time.Time(*f).Format("2006-01-02 15:04:05")
					{{ endswitch }}
				}
		{{ endswitch }}
		func (f *{{ tab.ModelName}}{{ col.FieldName}}) tableName() string {
			return "{{ tab.TableName }}"
		}
	{{ endfor }}
{{ endfor }}
`

func TestFmt(t *testing.T) {
	l, err := parseStmt(src)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range l {
		fmt.Println(s.src)
	}
}
