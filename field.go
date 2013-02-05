package pgsql

import (
	"fmt"
	"strconv"
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

type Selection []interface{}

type Field struct {
	Name        string
	flags       flag
	Default     Sqler
	Type        Type
	Table       *Table
	ForeignKey  *Field
	Selection   Selection
	Validations []Validator
}

func NewField(name string, options ...interface{}) *Field {
	ø := &Field{
		Name:        name,
		flags:       hasDefaults,
		Validations: []Validator{},
	}
	ø.Add(options...)
	return ø
}

func (ø *Field) InSelection(value interface{}) bool {
	if ø.Selection == nil {
		return true
	}
	asString := fmt.Sprintf("%v", value)
	for _, s := range ø.Selection {
		if fmt.Sprintf("%v", s) == asString {
			return true
		}
	}
	return false
}

func (ø *Field) IsValid(value interface{}) bool {
	valString := ToString(value)
	if value == nil && ø.Is(NullAllowed) {
		return true
	}
	if !ø.InSelection(value) {
		return false
	}
	switch ø.Type {
	case IntType:
		_, err := strconv.ParseInt(valString, 10, 32)
		if err == nil {
			return true
		}
	case FloatType:
		_, err := strconv.ParseFloat(valString, 32)
		if err == nil {
			return true
		}
	default:
		if IsVarChar(ø.Type) {
			return len(valString) <= int(ø.Type)
		} else {
			return true
		}
	}
	return false
}

func (ø *Field) Add(options ...interface{}) {
	for _, option := range options {
		switch v := option.(type) {
		case *Table:
			ø.Table = v
		case Type:
			ø.Type = v
		case flag:
			ø.flags = ø.flags | v
		case *Field:
			ø.ForeignKey = v
		case Selection:
			ø.Selection = v
		default:
			if val, ok := v.(Validator); ok {
				ø.Validations = append(ø.Validations, val)
				continue
			}
			if sqler, ok := v.(Sqler); ok {
				ø.Default = sqler
			} else {
				panic("unknown type for field " + fmt.Sprintf("%v\n", v))
			}
		}
	}
}

// checks if a given flag is set, e.g.
//
// 	Is(NullAllowed)
//
// checks is null is allowed
func (ø *Field) Is(f flag) bool {
	return ø.flags&f != 0
}

func (ø *Field) Sql() SqlType {
	if ø.Table == nil {
		return Sql(fmt.Sprintf(`"%s"`, ø.Name))
	}
	return Sql(fmt.Sprintf(`%s."%s"`, ø.Table.Sql(), ø.Name))
}
