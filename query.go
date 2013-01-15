package pgdb

import (
	"fmt"
	"strings"
)

type Limit int

func (ø Limit) Sql() Sql {
	if ø == 0 {
		return Sql("")
	}
	return Sql(fmt.Sprintf("\nLIMIT %v", ø))
}

type Offset int

func (ø Offset) Sql() (s Sql) {
	if ø == 0 {
		s = Sql("")
	} else {
		s = Sql(fmt.Sprintf("\nOffset %v", ø))
	}
	return
}

type Query interface {
	Sql() (Sql, error)
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
func (ø Placeholder) Sql() Sql {
	return Sql(fmt.Sprintf("$%v", ø))
}

type Comparer struct {
	A    Sqler
	B    Sqler
	Sign string
}

func (ø *Comparer) Sql() Sql {
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

func (ø *InComparer) Sql() Sql {
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

func (ø *innerWhere) Sql() (s Sql) {
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

func (ø Where) Sql() (s Sql) {
	if ø.Inner == nil {
		s = Sql("")
	} else {
		s = Sql("\nWHERE \n\t" + ø.Inner.Sql())
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
	TableStruct *TableStruct
	Sets        []map[*FieldStruct]interface{}
}

func Insert(table *TableStruct, first_row Set, rows ...Set) Query {
	i := &InsertQuery{
		TableStruct: table,
		Sets:        []map[*FieldStruct]interface{}{(&first_row).Map()},
	}
	for _, r := range rows {
		i.Sets = append(i.Sets, (&r).Map())
	}
	return i
}

func (ø *InsertQuery) fieldsAndValues() (fields string, values string, err error) {
	fi := []string{}
	va := []string{}
	for k, _ := range ø.Sets[0] {
		fi = append(fi, string(k.Sql()))
	}

	for _, r := range ø.Sets {
		ro := []string{}
		for k, v := range r {
			//fi = append(fi, string(k.Sql()))
			if k.Is(NullAllowed) && v == nil {
				ro = append(ro, "null")
				continue
			}
			sql, e := k.Type.Escape(v)
			if e != nil {
				err = e
				return
			}
			ro = append(ro, string(sql))
		}
		va = append(va, "("+strings.Join(ro, ", ")+")")
	}
	fields = strings.Join(fi, ",")
	values = strings.Join(va, ",\n\t")
	return
}

func (ø *InsertQuery) Sql() (s Sql, err error) {
	t := ø.TableStruct
	currval := Sql(fmt.Sprintf("SELECT\n\tcurrval('%s')", t.PrimaryKeySeq))
	//SELECT currval('\"#{@table.schema.name}\".\"#{@table.primary_key_seq}\"') as id;"
	fi, va, err := ø.fieldsAndValues()
	if err != nil {
		return
	}
	s = Sql(fmt.Sprintf("INSERT INTO \n\t%s (%s) \nVALUES \n\t%s;\n%s", t.Sql(), fi, va, (&As{currval, "id"}).Sql()))
	return
}

type UpdateQuery struct {
	TableStruct *TableStruct
	Where       *Where
	Limit       Limit
	Set         map[*FieldStruct]interface{}
}

func Update(table *TableStruct, options ...interface{}) Query {
	u := &UpdateQuery{
		Limit:       Limit(0),
		TableStruct: table,
		Where:       &Where{},
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
		}
	}
	return u
}

func (ø *UpdateQuery) setString() (set string, err error) {
	sets := []string{}
	for k, v := range ø.Set {
		var valstr Sql
		if k.Is(NullAllowed) && v == nil {
			valstr = Sql("null")
		} else {
			valstr, err = k.Type.Escape(v)
			if err != nil {
				return
			}
		}
		sets = append(sets, fmt.Sprintf("%s = %s", k.Sql(), valstr))
	}
	set = strings.Join(sets, ",\n\t")
	return
}

func (ø *UpdateQuery) Sql() (s Sql, err error) {
	t := ø.TableStruct
	sets, err := ø.setString()
	if err != nil {
		return
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

type DeleteQuery struct {
	TableStruct *TableStruct
	Where       *Where
	Limit       Limit
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
		case *TableStruct:
			d.TableStruct = v
		}
	}
	return d
}

func (ø *DeleteQuery) Sql() (s Sql, err error) {
	s = Sql(
		fmt.Sprintf(
			"DELETE \n\t* \nFROM \n\t%s %s %s",
			ø.TableStruct.Sql(),
			ø.Where.Sql(),
			ø.Limit.Sql()))
	return
}

type Direction bool

var ASC = Direction(true)
var DESC = Direction(false)

func (ø Direction) Sql() (s Sql) {
	if ø {
		s = Sql("ASC")
	} else {
		s = Sql("DESC")
	}
	return
}

type OrderBy struct {
	*FieldStruct
	Direction Direction
}

func (ø *OrderBy) Sql() Sql {
	return Sql(ø.FieldStruct.Sql() + " " + ø.Direction.Sql())
}

type GroupBy []*FieldStruct

func (ø GroupBy) Sql() Sql {
	g := []string{}
	for _, f := range ø {
		g = append(g, string(f.Sql()))
	}
	return Sql("\nGROUP BY " + strings.Join(g, ","))
}

type Distinct bool

func (ø Distinct) Sql() (s Sql) {
	if ø {
		s = Sql("DISTINCT")
	} else {
		s = Sql("")
	}
	return
}

type SelectQuery struct {
	Distinct              Distinct
	TableStruct           Sqler
	Where                 *Where
	Limit                 Limit
	Joins                 []*Join
	FieldStructs          []*FieldStruct
	FieldStructsWithAlias []*As
	Offset                Offset
	OrderBy               []*OrderBy
	GroupBy               GroupBy
}

func Select(options ...interface{}) Query {
	s := &SelectQuery{
		Distinct:              Distinct(false),
		Joins:                 []*Join{},
		FieldStructs:          []*FieldStruct{},
		FieldStructsWithAlias: []*As{},
		Limit:                 Limit(0),
		Where:                 &Where{},
		OrderBy:               []*OrderBy{},
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
		case *FieldStruct:
			s.FieldStructs = append(s.FieldStructs, v)
		case *As:
			s.FieldStructsWithAlias = append(s.FieldStructsWithAlias, v)
		case Join:
			s.Joins = append(s.Joins, &v)
		case FieldStruct:
			s.FieldStructs = append(s.FieldStructs, &v)
		case As:
			s.FieldStructsWithAlias = append(s.FieldStructsWithAlias, &v)
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
				s.TableStruct = sqler
			} else {
				panic("unknown select option " + fmt.Sprintf("%#v", v))
			}
		}
	}
	if s.TableStruct == nil {
		panic("no table to select from")
	}
	return s
}

func (ø *SelectQuery) fieldstr() (s string) {
	f := []string{}

	for _, field := range ø.FieldStructs {
		f = append(f, string(field.Sql()))
	}

	for _, alias := range ø.FieldStructsWithAlias {
		f = append(f, alias.Sql())
	}

	s = strings.Join(f, ", \n\t")
	return
}

func (ø *SelectQuery) joins() (s Sql) {
	if len(ø.Joins) == 0 {
		return Sql("")
	}
	return Sql("")
}

func (ø *SelectQuery) group_by() (s Sql) {
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
func (ø *SelectQuery) Sql() (s Sql, err error) {
	s = Sql(
		fmt.Sprintf(
			"SELECT %s \n\t%s \nFROM \n\t%s %s %s %s %s %s",
			ø.Distinct.Sql(),
			ø.fieldstr(),
			ø.TableStruct.Sql(),
			ø.joins(),
			ø.Where.Sql(),
			ø.group_by(),
			ø.Limit.Sql(),
			ø.Offset.Sql()))
	return
}

type Set []interface{}

func (ø Set) Map() (m map[*FieldStruct]interface{}) {
	m = map[*FieldStruct]interface{}{}
	for i := 0; i < len(ø); i = i + 2 {
		m[ø[i].(*FieldStruct)] = ø[i+1]
	}
	return
}
