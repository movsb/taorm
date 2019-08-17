package taorm

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/movsb/taorm/filter"
)

// _Where ...
type _Where struct {
	query string
	args  []interface{}
}

func (w *_Where) rebuild() (query string, args []interface{}) {
	sb := bytes.NewBuffer(nil)
	var i int
	for _, c := range w.query {
		switch c {
		case '?':
			if i >= len(w.args) {
				panic(fmt.Errorf("err where args count"))
			}
			value := reflect.ValueOf(w.args[i])
			if value.Kind() == reflect.Slice {
				n := value.Len()
				sb.WriteString(createSQLInMarks(n))
				for j := 0; j < n; j++ {
					args = append(args, value.Index(j).Interface())
				}
			} else {
				sb.WriteByte('?')
				args = append(args, w.args[i])
			}
			i++
		default:
			sb.WriteRune(c)
		}
	}
	if i != len(w.args) {
		panic(fmt.Errorf("err where args count"))
	}
	return sb.String(), args
}

type _RawQuery struct {
	query string
	args  []interface{}
}

type _Filter struct {
	filter string
	mapper filter.Mapper
}

// Stmt is an SQL statement.
type Stmt struct {
	db              *DB
	raw             _RawQuery // not set if query == ""
	filter          _Filter
	model           interface{}
	fromTable       interface{}
	info            *_StructInfo
	tableNames      []string
	innerJoinTables []string
	fields          []string
	ands            []_Where
	groupBy         string
	orderBy         string
	limit           int64
	offset          int64
}

// From ...
// table can be either string or struct.
func (s *Stmt) From(table interface{}) *Stmt {
	switch typed := table.(type) {
	case string:
		s.tableNames = append(s.tableNames, typed)
	default:
		name, err := s.tryFindTableName(table)
		if err != nil {
			panic(WrapError(err))
		}
		s.tableNames = append(s.tableNames, name)
	}
	return s
}

// InnerJoin ...
func (s *Stmt) InnerJoin(table interface{}, on string) *Stmt {
	name := ""
	switch typed := table.(type) {
	case string:
		name = typed
	default:
		n, err := s.tryFindTableName(typed)
		if err != nil {
			panic(WrapError(err))
		}
		name = n
	}

	q := " INNER JOIN " + name
	if on != "" {
		q += " ON " + on
	}
	s.innerJoinTables = append(s.innerJoinTables, q)
	return s
}

// Select ...
func (s *Stmt) Select(fields string) *Stmt {
	if len(fields) > 0 {
		s.fields = append(s.fields, fields)
	}
	return s
}

// Where ...
func (s *Stmt) Where(query string, args ...interface{}) *Stmt {
	w := _Where{
		query: query,
		args:  args,
	}
	s.ands = append(s.ands, w)
	return s
}

// WhereIf ...
func (s *Stmt) WhereIf(cond bool, query string, args ...interface{}) *Stmt {
	if cond {
		s.Where(query, args...)
	}
	return s
}

// GroupBy ...
func (s *Stmt) GroupBy(groupBy string) *Stmt {
	s.groupBy = groupBy
	return s
}

// OrderBy ...
// TODO multiple orderbys
func (s *Stmt) OrderBy(orderBy string) *Stmt {
	s.orderBy = orderBy
	return s
}

// Limit ...
func (s *Stmt) Limit(limit int64) *Stmt {
	s.limit = limit
	return s
}

// Offset ...
func (s *Stmt) Offset(offset int64) *Stmt {
	s.offset = offset
	return s
}

// Filter ...
func (s *Stmt) Filter(expr string, mapper filter.Mapper) *Stmt {
	s.filter = _Filter{
		filter: expr,
		mapper: mapper,
	}
	return s
}

// noWheres returns true if no SQL conditions.
// Includes and, or.
func (s *Stmt) noWheres() bool {
	return len(s.ands) <= 0
}

func (s *Stmt) buildWheres() (string, []interface{}) {
	if s.model != nil {
		id, ok := s.info.getPrimaryKey(s.model)
		s.WhereIf(ok, "id=?", id)
	}

	if s.noWheres() {
		return "", nil
	}

	var args []interface{}
	sb := bytes.NewBuffer(nil)
	fw := func(format string, args ...interface{}) {
		sb.WriteString(fmt.Sprintf(format, args...))
	}
	sb.WriteString(" WHERE ")
	for i, w := range s.ands {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		query, xargs := w.rebuild()
		fw("(%s)", query)
		args = append(args, xargs...)
	}
	return sb.String(), args
}

func (s *Stmt) buildCreate() (*_StructInfo, string, []interface{}, error) {
	panicIf(len(s.tableNames) != 1, "model length is not 1")
	panicIf(s.raw.query != "", "cannot use raw here")
	info, err := getRegistered(s.model)
	if err != nil {
		return info, "", nil, err
	}
	args := info.ifacesOf(s.model)
	if len(args) == 0 {
		return info, "", nil, ErrNoFields
	}
	return info, info.insertstr, args, nil
}

func (s *Stmt) tryFindTableName(out interface{}) (string, error) {
	info, err := getRegistered(out)
	if err != nil {
		return "", err
	}
	if info.tableName == "" {
		return "", fmt.Errorf("trying to use auto-registered struct table name")
	}
	return info.tableName, nil
}

func (s *Stmt) buildSelect(out interface{}, isCount bool) (string, []interface{}, error) {
	if s.raw.query != "" {
		return s.raw.query, s.raw.args, nil
	}

	if s.filter.filter != "" {
		info, err := getRegistered(out)
		if err != nil {
			return "", nil, err
		}
		query, args, err := filter.Filter(
			func(field string) reflect.Type {
				return info.fields[field]._type // maybe not exist
			},
			s.filter.filter,
			s.filter.mapper,
		)
		if err != nil {
			return "", nil, err
		}
		s.WhereIf(query != "", query, args...)
		s.filter.filter = ""
	}

	if len(s.tableNames) == 0 {
		name, err := s.tryFindTableName(out)
		if err != nil {
			return "", nil, err
		}
		s.tableNames = append(s.tableNames, name)
	}

	panicIf(len(s.tableNames) == 0, "model is empty")

	var strFields string

	if isCount {
		strFields = "COUNT(1)"
	} else {
		fields := []string{}
		if len(s.fields) == 0 {
			if len(s.innerJoinTables) == 0 {
				fields = []string{"*"}
			} else {
				fields = []string{s.tableNames[0] + ".*"}
			}
		} else {
			if len(s.innerJoinTables) == 0 || len(s.fields) == 1 && s.fields[0] == "*" {
				fields = s.fields
			} else {
				for _, list := range s.fields {
					slice := strings.Split(list, ",")
					for _, field := range slice {
						index := strings.IndexByte(field, '.')
						if index == -1 {
							fields = append(fields, fmt.Sprintf("%s.%s", s.tableNames[0], field))
						} else {
							fields = append(fields, field)
						}
					}
				}
			}
		}
		strFields = strings.Join(fields, ",")
	}

	query := fmt.Sprintf(`SELECT %s FROM %s`, strFields, strings.Join(s.tableNames, ","))
	if len(s.innerJoinTables) > 0 {
		query += strings.Join(s.innerJoinTables, " ")
	}

	var args []interface{}

	whereQuery, whereArgs := s.buildWheres()
	query += whereQuery
	args = append(args, whereArgs...)

	query += s.buildGroupBy()
	if orderBy, err := s.buildOrderBy(); err != nil {
		return "", nil, err
	} else {
		if orderBy != "" {
			query += orderBy
		}
	}
	query += s.buildLimit()

	return query, args, nil
}

func (s *Stmt) buildUpdateMap(fields map[string]interface{}) (string, []interface{}, error) {
	panicIf(len(s.tableNames) == 0, "model is empty")
	panicIf(s.raw.query != "", "cannot use raw here")
	var args []interface{}
	query := fmt.Sprintf(`UPDATE %s SET `, strings.Join(s.tableNames, ","))

	var updates []string
	var values []interface{}

	if len(fields) == 0 {
		return "", nil, ErrNoFields
	}

	for field, value := range fields {
		if expr, ok := value.(Expr); ok {
			pair := fmt.Sprintf("%s=%s", field, string(expr))
			updates = append(updates, pair)
			continue
		}
		pair := fmt.Sprintf("%s=?", field)
		updates = append(updates, pair)
		values = append(values, value)
	}

	query += strings.Join(updates, ",")
	args = append(args, values...)

	whereQuery, whereArgs := s.buildWheres()
	query += whereQuery
	args = append(args, whereArgs...)

	query += s.buildLimit()

	return query, args, nil
}

func (s *Stmt) buildUpdateModel(model interface{}) (string, []interface{}, error) {
	panicIf(len(s.tableNames) == 0, "model is empty")
	panicIf(s.raw.query != "", "cannot use raw here")
	query := s.info.updatestr
	args := s.info.ifacesOf(model)
	whereQuery, whereArgs := s.buildWheres()
	query += whereQuery
	args = append(args, whereArgs...)
	return query, args, nil
}

func (s *Stmt) buildDelete() (string, []interface{}, error) {
	panicIf(len(s.tableNames) == 0, "model is empty")
	panicIf(s.raw.query != "", "cannot use raw here")
	var args []interface{}
	query := fmt.Sprintf(`DELETE FROM %s`, strings.Join(s.tableNames, ","))

	whereQuery, whereArgs := s.buildWheres()
	query += whereQuery
	args = append(args, whereArgs...)

	query += s.buildLimit()

	return query, args, nil
}

func (s *Stmt) buildGroupBy() (groupBy string) {
	if s.groupBy != "" {
		groupBy = fmt.Sprintf(` GROUP BY %s`, s.groupBy)
	}
	return
}

var regexpOrderBy = regexp.MustCompile(`^ *((\w+\.)?(\w+)) *(\w+)? *$`)

func (s *Stmt) buildOrderBy() (string, error) {
	orderBy := " ORDER BY "
	if s.orderBy == "" {
		return "", nil
	}
	parts := strings.Split(s.orderBy, ",")
	orderBys := []string{}
	for _, part := range parts {
		matches := regexpOrderBy.FindStringSubmatch(part)
		if len(matches) != 5 {
			return "", errors.New("invalid orderby")
		}
		table := matches[2]
		column := matches[1]
		order := matches[4]
		// avoid column ambiguous
		// "Error 1052: Column 'created_at' in order clause is ambiguous"
		if table == "" && len(s.tableNames)+len(s.innerJoinTables) > 1 {
			column = s.tableNames[0] + "." + column
		}
		if order != "" {
			column += " " + order
		}
		orderBys = append(orderBys, column)
	}
	orderBy += strings.Join(orderBys, ",")
	return orderBy, nil
}

func (s *Stmt) buildLimit() (limit string) {
	if s.limit > 0 {
		limit += fmt.Sprintf(" LIMIT %d", s.limit)
		if s.offset >= 0 {
			limit += fmt.Sprintf(" OFFSET %d", s.offset)
		}
	}
	return
}

// Create ...
func (s *Stmt) Create() error {
	info, query, args, err := s.buildCreate()
	if err != nil {
		return WrapError(err)
	}

	dumpSQL(query, args...)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return WrapError(err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return WrapError(err)
	}

	info.setPrimaryKey(s.model, id)

	return nil
}

// MustCreate ...
func (s *Stmt) MustCreate() {
	if err := s.Create(); err != nil {
		panic(err)
	}
}

// CreateSQL ...
func (s *Stmt) CreateSQL() string {
	_, query, args, err := s.buildCreate()
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}

// Find ...
func (s *Stmt) Find(out interface{}) error {
	query, args, err := s.buildSelect(out, false)
	if err != nil {
		return WrapError(err)
	}

	dumpSQL(query, args...)
	return ScanRows(out, s.db, query, args...)
}

// MustFind ...
func (s *Stmt) MustFind(out interface{}) {
	if err := s.Find(out); err != nil {
		panic(err)
	}
}

// FindSQL ...
func (s *Stmt) FindSQL() string {
	query, args, err := s.buildSelect(s.model, false)
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}

// Count ...
func (s *Stmt) Count(out interface{}) error {
	query, args, err := s.buildSelect(s.fromTable, true)
	if err != nil {
		return WrapError(err)
	}

	dumpSQL(query, args...)
	return ScanRows(out, s.db, query, args...)
}

// MustCount ...
func (s *Stmt) MustCount(out interface{}) {
	if err := s.Count(out); err != nil {
		panic(err)
	}
}

// CountSQL ...
func (s *Stmt) CountSQL() string {
	query, args, err := s.buildSelect(s.fromTable, true)
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}

func (s *Stmt) updateMap(fields M, anyway bool) error {
	query, args, err := s.buildUpdateMap(fields)
	if err != nil {
		if err == ErrNoFields {
			return nil
		}
		return err
	}

	if !anyway && s.noWheres() {
		return ErrNoWhere
	}

	dumpSQL(query, args...)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

func (s *Stmt) updateModel(model interface{}) error {
	query, args, err := s.buildUpdateModel(model)
	if err != nil {
		if err == ErrNoFields {
			return nil
		}
		return err
	}

	dumpSQL(query, args...)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

// UpdateMap ...
func (s *Stmt) UpdateMap(updates M) error {
	return WrapError(s.updateMap(updates, false))
}

// UpdateMapAnyway ...
func (s *Stmt) UpdateMapAnyway(updates M) error {
	return WrapError(s.updateMap(updates, true))
}

// UpdateModel ...
func (s *Stmt) UpdateModel(model interface{}) error {
	return WrapError(s.updateModel(model))
}

// MustUpdateMap ...
func (s *Stmt) MustUpdateMap(updates M) {
	if err := s.updateMap(updates, false); err != nil {
		panic(err)
	}
}

// MustUpdateMapAnyway ...
func (s *Stmt) MustUpdateMapAnyway(updates M) {
	if err := s.updateMap(updates, true); err != nil {
		panic(err)
	}
}

// UpdateMapSQL ...
func (s *Stmt) UpdateMapSQL(updates M) string {
	query, args, err := s.buildUpdateMap(updates)
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}

// UpdateModelSQL ...
func (s *Stmt) UpdateModelSQL(model interface{}) string {
	query, args, err := s.buildUpdateModel(model)
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}

func (s *Stmt) _delete(anyway bool) error {
	query, args, err := s.buildDelete()
	if err != nil {
		return err
	}

	if !anyway && s.noWheres() {
		return ErrNoWhere
	}

	dumpSQL(query, args...)

	_, err = s.db.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}

// Delete ...
func (s *Stmt) Delete() error {
	return WrapError(s._delete(false))
}

// DeleteAnyway ...
func (s *Stmt) DeleteAnyway() error {
	return WrapError(s._delete(true))
}

// MustDelete ...
func (s *Stmt) MustDelete() {
	if err := s.Delete(); err != nil {
		panic(err)
	}
}

// MustDeleteAnyway ...
func (s *Stmt) MustDeleteAnyway() {
	if err := s.DeleteAnyway(); err != nil {
		panic(err)
	}
}

// DeleteSQL ...
func (s *Stmt) DeleteSQL() string {
	query, args, err := s.buildDelete()
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}
