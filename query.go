package pgdb

import (
	"fmt"
	"strings"
)

type Limit int

func (ø Limit) Sql() SqlType {
	if ø == 0 {
		return Sql("")
	}
	return Sql(fmt.Sprintf("\nLIMIT %v", ø))
}

type Offset int

func (ø Offset) Sql() (s SqlType) {
	if ø == 0 {
		s = Sql("")
	} else {
		s = Sql(fmt.Sprintf("\nOffset %v", ø))
	}
	return
}

type Query interface {
	Sql() SqlType
	String() string
}

const (
	LeftJoin JoinType = iota
	RightJoin
	InnerJoin
	FullJoin
)

type JoinType int
type Placeholder int

// you can use Placeholder(1), as value to update insert etc. in order
// to get a prepare statement, that you may execute later on
func (ø Placeholder) Sql() SqlType {
	return Sql(fmt.Sprintf("$%v", ø))
}

type Comparer struct {
	A    Sqler
	B    Sqler
	Sign string
}

func (ø *Comparer) Sql() SqlType {
	return Sql(fmt.Sprintf("%s %s %s", ø.A.Sql(), ø.Sign, ø.B.Sql()))
}

func Equals(a Sqler, b Sqler) *Comparer {
	return &Comparer{a, b, "="}
}

func EqualsNot(a Sqler, b Sqler) *Comparer {
	return &Comparer{a, b, "!="}
}

func GreaterThan(a Sqler, b Sqler) *Comparer {
	return &Comparer{a, b, ">"}
}

func LessThan(a Sqler, b Sqler) *Comparer {
	return &Comparer{a, b, "<"}
}

func GreaterThanEqual(a Sqler, b Sqler) *Comparer {
	return &Comparer{a, b, ">="}
}

func LessThanEqual(a Sqler, b Sqler) *Comparer {
	return &Comparer{a, b, "<="}
}

type InComparer struct {
	A  Sqler
	Bs []Sqler
}

func In(a Sqler, bs ...Sqler) *InComparer {
	return &InComparer{a, bs}
}

func (ø *InComparer) Sql() SqlType {
	bs := []string{}
	for _, b := range ø.Bs {
		bs = append(bs, string(b.Sql()))
	}
	return Sql(fmt.Sprintf("%s In(%s)", ø.A.Sql(), strings.Join(bs, ", ")))
}

type innerWhere struct {
	Conditions []Sqler
	Sign       string
}

func Or(sqls ...Sqler) *innerWhere {
	return &innerWhere{sqls, "OR"}
}

func And(sqls ...Sqler) *innerWhere {
	return &innerWhere{sqls, "AND"}
}

func (ø *innerWhere) Sql() (s SqlType) {
	if len(ø.Conditions) == 0 {
		s = Sql("")
	} else {
		w := []string{}
		for _, cond := range ø.Conditions {
			w = append(w, "("+string(cond.Sql())+")")
		}
		s = Sql(strings.Join(w, " "+ø.Sign+" "))
	}
	return
}

type Where struct {
	Inner Sqler
}

func (ø Where) Sql() (s SqlType) {
	if ø.Inner == nil {
		s = Sql("")
	} else {
		s = Sql("\nWHERE \n\t" + ø.Inner.Sql().String())
	}
	return
}

type Join struct {
	A    Sqler
	B    Sqler
	Type JoinType
	On   *Comparer
}

type As struct {
	Sqler Sqler
	As    string
}

func (ø *As) Sql() string {
	return fmt.Sprintf("%s as %s", ø.Sqler.Sql(), ø.As)
}

type InsertQuery struct {
	Table *Table
	Sets  []map[*Field]interface{}
}

func InsertMap(table *Table, m map[*Field]interface{}) Query {
	return &InsertQuery{
		Table: table,
		Sets:  []map[*Field]interface{}{m},
	}
}

func Insert(table *Table, first_row Set, rows ...Set) Query {
	i := &InsertQuery{
		Table: table,
		Sets:  []map[*Field]interface{}{(&first_row).Map()},
	}
	for _, r := range rows {
		i.Sets = append(i.Sets, (&r).Map())
	}
	return i
}

func (ø *InsertQuery) fieldsAndValues() (fields string, values string, err error) {
	fi := []string{}
	va := []string{}
	fieldorder := []*Field{}
	for k, _ := range ø.Sets[0] {
		fieldorder = append(fieldorder, k)
		fi = append(fi, string(k.Sql()))
	}

	for _, r := range ø.Sets {
		ro := []string{}
		for _, k := range fieldorder {
			v := r[k]
			//fi = append(fi, string(k.Sql()))
			if k.Is(NullAllowed) && v == nil {
				ro = append(ro, "null")
				continue
			}
			//sql, e := k.Type.Escape(v)

			tv := TypedValue{PgType: k.Type}
			e := Convert(v, &tv)
			if e != nil {
				err = e
				return
			}

			sql := tv.Sql()
			/*
				if e != nil {
					err = fmt.Errorf("error in %v: %s", k.Name, e)
					return
				}
			*/
			//fmt.Println(sql)
			ro = append(ro, string(sql))
			//ro = append(ro, string(""))
		}
		rs := strings.Join(ro, ", ")
		va = append(va, "("+rs+")")
	}
	fields = strings.Join(fi, ",")
	values = strings.Join(va, ",\n\t")
	return
}

func (ø *InsertQuery) Sql() (s SqlType) {
	t := ø.Table
	currval := Sql(fmt.Sprintf("SELECT\n\tcurrval('%s')", t.PrimaryKeySeq))
	//SELECT currval('\"#{@table.schema.name}\".\"#{@table.primary_key_seq}\"') as id;"
	fi, va, err := ø.fieldsAndValues()
	if err != nil {
		panic(err)
	}
	s = Sql(fmt.Sprintf("INSERT INTO \n\t%s (%s) \nVALUES \n\t%s;\n%s", t.Sql(), fi, va, (&As{currval, "id"}).Sql()))
	return
}

func (ø *InsertQuery) String() string {
	return ø.Sql().String()
}

type UpdateQuery struct {
	Table *Table
	Where *Where
	Limit Limit
	Set   map[*Field]interface{}
}

func Update(table *Table, options ...interface{}) Query {
	u := &UpdateQuery{
		Limit: Limit(0),
		Table: table,
		Where: &Where{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *Where:
			u.Where = v
		case Where:
			u.Where = &v
		case Limit:
			u.Limit = v
		case Set:
			u.Set = (&v).Map()
		case *Set:
			u.Set = v.Map()
		case map[*Field]interface{}:
			u.Set = v
		}
	}
	return u
}

func (ø *UpdateQuery) setString() (set string, err error) {
	sets := []string{}
	for k, v := range ø.Set {
		var valstr SqlType
		if k.Is(NullAllowed) && v == nil {
			valstr = Sql("null")
		} else {

			tv := TypedValue{PgType: k.Type}
			e := Convert(v, &tv)
			if e != nil {
				err = e
				return
			}

			valstr = tv.Sql()
			//valstr, err = k.Type.Escape(v)
			/*
				if err != nil {
					return
				}
			*/
		}
		sets = append(sets, fmt.Sprintf("%s = %s", k.Sql(), valstr))
	}
	set = strings.Join(sets, ",\n\t")
	return
}

func (ø *UpdateQuery) Sql() (s SqlType) {
	t := ø.Table
	sets, err := ø.setString()
	if err != nil {
		panic(err)
	}
	s = Sql(
		fmt.Sprintf(
			"UPDATE \n\t%s \nSET \n\t%s %s %s",
			t.Sql(),
			sets,
			ø.Where.Sql(),
			ø.Limit.Sql()))
	return
}

func (ø *UpdateQuery) String() string {
	return ø.Sql().String()
}

type DeleteQuery struct {
	Table *Table
	Where *Where
	Limit Limit
}

func Delete(options ...interface{}) Query {
	d := &DeleteQuery{
		Where: &Where{},
		Limit: Limit(0)}
	for _, option := range options {
		switch v := option.(type) {
		case *Where:
			d.Where = v
		case Where:
			d.Where = &v
		case Limit:
			d.Limit = v
		case *Table:
			d.Table = v
		}
	}
	return d
}

func (ø *DeleteQuery) Sql() (s SqlType) {
	s = Sql(
		fmt.Sprintf(
			"DELETE \n\t* \nFROM \n\t%s %s %s",
			ø.Table.Sql(),
			ø.Where.Sql(),
			ø.Limit.Sql()))
	return
}

func (ø *DeleteQuery) String() string {
	return ø.Sql().String()
}

type Direction bool

var ASC = Direction(true)
var DESC = Direction(false)

func (ø Direction) Sql() (s SqlType) {
	if ø {
		s = Sql("ASC")
	} else {
		s = Sql("DESC")
	}
	return
}

type OrderBy struct {
	*Field
	Direction Direction
}

func (ø *OrderBy) Sql() SqlType {
	return Sql(ø.Field.Sql().String() + " " + ø.Direction.Sql().String())
}

type GroupBy []*Field

func (ø GroupBy) Sql() SqlType {
	g := []string{}
	for _, f := range ø {
		g = append(g, string(f.Sql()))
	}
	return Sql("\nGROUP BY " + strings.Join(g, ","))
}

type Distinct bool

func (ø Distinct) Sql() (s SqlType) {
	if ø {
		s = Sql("DISTINCT")
	} else {
		s = Sql("")
	}
	return
}

type SelectQuery struct {
	Distinct        Distinct
	Table           Sqler
	Where           *Where
	Limit           Limit
	Joins           []*Join
	Fields          []*Field
	FieldsWithAlias []*As
	Offset          Offset
	OrderBy         []*OrderBy
	GroupBy         GroupBy
}

func Select(options ...interface{}) Query {
	s := &SelectQuery{
		Distinct:        Distinct(false),
		Joins:           []*Join{},
		Fields:          []*Field{},
		FieldsWithAlias: []*As{},
		Limit:           Limit(0),
		Where:           &Where{},
		OrderBy:         []*OrderBy{},
	}
	for _, option := range options {
		switch v := option.(type) {
		case *Where:
			s.Where = v
		case Where:
			s.Where = &v
		case Limit:
			s.Limit = v
		case *Join:
			s.Joins = append(s.Joins, v)
		case *Field:
			s.Fields = append(s.Fields, v)
		case *As:
			s.FieldsWithAlias = append(s.FieldsWithAlias, v)
		case Join:
			s.Joins = append(s.Joins, &v)
		case Field:
			s.Fields = append(s.Fields, &v)
		case As:
			s.FieldsWithAlias = append(s.FieldsWithAlias, &v)
		case Offset:
			s.Offset = v
		case *OrderBy:
			s.OrderBy = append(s.OrderBy, v)
		case OrderBy:
			s.OrderBy = append(s.OrderBy, &v)
		case GroupBy:
			s.GroupBy = v
		default:
			if sqler, ok := v.(Sqler); ok {
				s.Table = sqler
			} else {
				panic("unknown select option " + fmt.Sprintf("%#v", v))
			}
		}
	}
	if s.Table == nil {
		panic("no table to select from")
	}
	return s
}

func (ø *SelectQuery) fieldstr() (s string) {
	f := []string{}

	for _, field := range ø.Fields {
		f = append(f, string(field.Sql()))
	}

	for _, alias := range ø.FieldsWithAlias {
		f = append(f, alias.Sql())
	}

	s = strings.Join(f, ", \n\t")
	return
}

func (ø *SelectQuery) joins() (s SqlType) {
	if len(ø.Joins) == 0 {
		return Sql("")
	}
	return Sql("")
}

func (ø *SelectQuery) group_by() (s SqlType) {
	if len(ø.GroupBy) == 0 {
		return Sql("")
	}
	str := []string{}
	for _, g := range ø.GroupBy {
		str = append(str, string(g.Sql()))
	}
	return Sql("\nGROUP BY " + strings.Join(str, ", "))
}

/*
	SELECT #{distinct}
		#{fields}
	FROM
		#{table}
	#{joins}
	#{left_joins}
	#{where}
	#{group_by}
	#{order_by}
	#{limit}
	#{offset}"
*/
func (ø *SelectQuery) Sql() (s SqlType) {
	s = Sql(
		fmt.Sprintf(
			"SELECT %s \n\t%s \nFROM \n\t%s %s %s %s %s %s",
			ø.Distinct.Sql(),
			ø.fieldstr(),
			ø.Table.Sql(),
			ø.joins(),
			ø.Where.Sql(),
			ø.group_by(),
			ø.Limit.Sql(),
			ø.Offset.Sql()))
	return
}

func (ø *SelectQuery) String() string {
	return ø.Sql().String()
}

type Set []interface{}

func (ø Set) Map() (m map[*Field]interface{}) {
	m = map[*Field]interface{}{}
	for i := 0; i < len(ø); i = i + 2 {
		m[ø[i].(*Field)] = ø[i+1]
	}
	return
}
