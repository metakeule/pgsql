package pgdb

import (
	"fmt"
)

type flag int

type Validator interface {
	Valid(interface{}) bool
}

const (
	_                    = iota
	hasDefaults     flag = 1 << iota
	NullAllowed          // field may have null values
	PrimaryKey           // field is primary key
	Indexed              // field is indexed
	Unique               // field is unique
	Serial               // field is a serial field
	OnDeleteCascade      // fkey is on delete cascade (default: restrict)
)

type FieldStruct struct {
	Name        string
	flags       flag
	Default     Sqler
	Type        Type
	TableStruct *TableStruct
	ForeignKey  *FieldStruct
	Validations []Validator
}

func Field(name string, options ...interface{}) *FieldStruct {
	t := &FieldStruct{
		Name:        name,
		flags:       hasDefaults,
		Validations: []Validator{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *TableStruct:
			t.TableStruct = v
		case Type:
			t.Type = v
		case flag:
			t.flags = t.flags | v
		case *FieldStruct:
			t.ForeignKey = v
		default:
			if val, ok := v.(Validator); ok {
				t.Validations = append(t.Validations, val)
				continue
			}
			if sqler, ok := v.(Sqler); ok {
				t.Default = sqler
			} else {
				panic("unknown type for field " + fmt.Sprintf("%v\n", v))
			}
		}
	}
	return t
}

// checks if a given flag is set, e.g.
//
// 	Is(NullAllowed)
//
// checks is null is allowed
func (ø *FieldStruct) Is(f flag) bool {
	return ø.flags&f != 0
}

func (ø *FieldStruct) Sql() Sql {
	if ø.TableStruct == nil {
		return Sql(fmt.Sprintf(`"%s"`, ø.Name))
	}
	return Sql(fmt.Sprintf(`%s."%s"`, ø.TableStruct.Sql(), ø.Name))
}
