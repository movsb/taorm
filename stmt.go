package taorm

import (
	"bytes"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// _Where ...
type _Where struct {
	query string
	args  []interface{}
}

func (w _Where) build() (query string, args []interface{}) {
	sb := bytes.NewBuffer(nil)
	sb.Grow(len(query)) // should we reserve capacity for slice too?
	var i int
	for _, c := range w.query {
		switch c {
		case '?':
			if i >= len(w.args) {
				panic(fmt.Errorf("err where args count"))
			}
			value := reflect.ValueOf(w.args[i])
			if value.Kind() == reflect.Slice {
				sliceValueKind := value.Type().Elem().Kind()
				switch sliceValueKind {
				case reflect.Uint8:
					// 对 []byte 特殊处理。
					sb.WriteByte('?')
					args = append(args, w.args[i])
				default:
					n := value.Len()
					marks := createSQLInMarks(n)
					sb.WriteString(marks)
					for j := 0; j < n; j++ {
						args = append(args, value.Index(j).Interface())
					}
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

// _Expr is a raw SQL expression.
//
// e.g.: `UPDATE sth SET left = right`, here `right` is the expression.
//
// TODO expr args cannot be slice.
type _Expr _Where

// Expr creates an expression for Update* operations.
func Expr(expr string, args ...interface{}) _Expr {
	return _Expr{
		query: expr,
		args:  args,
	}
}

type _RawQuery struct {
	query string
	args  []interface{}
}

// Stmt is an SQL statement.
type Stmt struct {
	db         *DB
	raw        _RawQuery // not set if query == ""
	model      interface{}
	fromTable  interface{}
	info       *_StructInfo
	tableNames []string
	joinTables []_Join
	fields     []string
	ands       []_Where
	groupBy    string
	having     string
	orderBy    string
	limit      int64
	offset     int64
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

type _Join struct {
	by    string
	table string
	where _Where
}

func (s *Stmt) InnerJoin(table any, on string, args ...any) *Stmt {
	return s.join(`INNER JOIN`, table, on, args...)
}

func (s *Stmt) LeftJoin(table any, on string, args ...any) *Stmt {
	return s.join(`LEFT JOIN`, table, on, args...)
}

func (s *Stmt) RightJoin(table any, on string, args ...any) *Stmt {
	return s.join(`RIGHT JOIN`, table, on, args...)
}

func (s *Stmt) join(by string, table any, on string, args ...any) *Stmt {
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

	s.joinTables = append(s.joinTables, _Join{
		by:    by,
		table: name,
		where: _Where{
			query: on,
			args:  args,
		},
	})

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

// Having ...
func (s *Stmt) Having(having string) *Stmt {
	s.having = having
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
	sb.WriteString(" WHERE ")
	for i, w := range s.ands {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		query, xargs := w.build()
		sb.WriteString("(" + query + ")")
		args = append(args, xargs...)
	}
	return sb.String(), args
}

func (s *Stmt) buildJoins() (string, []any) {
	if len(s.joinTables) <= 0 {
		return "", nil
	}

	var args []any
	sb := bytes.NewBuffer(nil)
	for _, j := range s.joinTables {
		fmt.Fprintf(sb, ` %s `, j.by)
		sb.WriteString(j.table)
		sb.WriteString(" ON ")
		query, xargs := j.where.build()
		sb.WriteString(query)
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
	var query string
	if pk, ok := info.getPrimaryKey(s.model); ok {
		args = append([]any{pk}, args...)
		query = info.insertIdStr
	} else {
		query = info.insertstr
	}
	return info, query, args, nil
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
			if len(s.joinTables) == 0 {
				fields = []string{"*"}
			} else {
				fields = []string{s.tableNames[0] + ".*"}
			}
		} else {
			if len(s.joinTables) == 0 || len(s.fields) == 1 && s.fields[0] == "*" {
				fields = s.fields
			} else {
				for _, list := range s.fields {
					slice := strings.Split(list, ",")
					for _, field := range slice {
						index := strings.IndexByte(field, '.')
						if index == -1 {
							f := s.tableNames[0] + "." + field
							fields = append(fields, f)
						} else {
							fields = append(fields, field)
						}
					}
				}
			}
		}
		strFields = strings.Join(fields, ",")
	}

	var args []interface{}

	query := `SELECT ` + strFields + ` FROM ` + strings.Join(s.tableNames, ",")
	if len(s.joinTables) > 0 {
		q, a := s.buildJoins()
		query += q
		args = append(args, a...)
	}

	whereQuery, whereArgs := s.buildWheres()
	query += whereQuery
	args = append(args, whereArgs...)

	query += s.buildGroupBy()
	query += s.buildHaving()

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
	query := `UPDATE ` + strings.Join(s.tableNames, ",") + ` SET `

	if len(fields) == 0 {
		return "", nil, ErrNoFields
	}

	updates := make([]string, 0, len(fields))
	args := make([]interface{}, 0, len(fields))

	for field, value := range fields {
		switch tv := value.(type) {
		case _Expr:
			eq, ea := _Where(tv).build()
			pair := field + "=" + eq
			updates = append(updates, pair)
			args = append(args, ea...)
		default:
			pair := field + "=?"
			updates = append(updates, pair)
			args = append(args, value)
		}
	}

	query += strings.Join(updates, ",")

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
	query := `DELETE FROM ` + strings.Join(s.tableNames, ",")

	whereQuery, whereArgs := s.buildWheres()
	query += whereQuery
	args = append(args, whereArgs...)

	query += s.buildLimit()

	return query, args, nil
}

func (s *Stmt) buildGroupBy() (groupBy string) {
	if s.groupBy != "" {
		groupBy = ` GROUP BY ` + s.groupBy
	}
	return
}

func (s *Stmt) buildHaving() (having string) {
	if s.having != `` {
		having = ` HAVING ` + s.having
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
		if !regexpOrderBy.MatchString(part) {
			return ``, fmt.Errorf(`invalid order_by: %s`, part)
		}
		orderBys = append(orderBys, part)

		// these are for automatically adding table names to fields in order_by etc.
		// they are commented out because of custom field name doesn't belong to some table.
		// currently I don't know how to handle this correctly.
		//
		// matches := regexpOrderBy.FindStringSubmatch(part)
		// if len(matches) != 5 {
		// 	return "", errors.New("invalid orderby")
		// }
		// table := matches[2]
		// column := matches[1]
		// order := matches[4]
		// // avoid column ambiguous
		// // "Error 1052: Column 'created_at' in order clause is ambiguous"
		// if table == "" && len(s.tableNames)+len(s.innerJoinTables) > 1 {
		// 	column = s.tableNames[0] + "." + column
		// }
		// if order != "" {
		// 	column += " " + order
		// }
		// orderBys = append(orderBys, column)
	}
	orderBy += strings.Join(orderBys, ",")
	return orderBy, nil
}

func (s *Stmt) buildLimit() (limit string) {
	if s.limit > 0 {
		limit += ` LIMIT ` + fmt.Sprint(s.limit)
		if s.offset >= 0 {
			limit += ` OFFSET ` + fmt.Sprint(s.offset)
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

func (s *Stmt) FindSQL() string {
	query, args, err := s.buildSelect(s.model, false)
	if err != nil {
		panic(WrapError(err))
	}
	return strSQL(query, args...)
}

// FindSQL ...
func (s *Stmt) FindSQLRaw() string {
	query, _, err := s.buildSelect(s.model, false)
	if err != nil {
		panic(WrapError(err))
	}
	return query
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

func (s *Stmt) updateMap(fields M, anyway bool) (sql.Result, error) {
	if len(fields) == 0 {
		return nil, ErrNoFields
	}

	query, args, err := s.buildUpdateMap(fields)
	if err != nil {
		return nil, err
	}

	if !anyway && s.noWheres() {
		return nil, ErrNoWhere
	}

	dumpSQL(query, args...)

	res, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Stmt) updateModel(model interface{}) (sql.Result, error) {
	query, args, err := s.buildUpdateModel(model)
	if err != nil {
		return nil, err
	}

	dumpSQL(query, args...)

	res, err := s.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// UpdateMap ...
func (s *Stmt) UpdateMap(updates M) (sql.Result, error) {
	res, err := s.updateMap(updates, false)
	return res, WrapError(err)
}

// UpdateMapAnyway ...
func (s *Stmt) UpdateMapAnyway(updates M) (sql.Result, error) {
	res, err := s.updateMap(updates, true)
	return res, WrapError(err)
}

// UpdateModel ...
func (s *Stmt) UpdateModel(model interface{}) (sql.Result, error) {
	res, err := s.updateModel(model)
	return res, WrapError(err)
}

// MustUpdateMap ...
func (s *Stmt) MustUpdateMap(updates M) sql.Result {
	res, err := s.updateMap(updates, false)
	if err != nil {
		panic(err)
	}
	return res
}

// MustUpdateMapAnyway ...
func (s *Stmt) MustUpdateMapAnyway(updates M) sql.Result {
	res, err := s.updateMap(updates, true)
	if err != nil {
		panic(err)
	}
	return res
}

// MustUpdateModel ...
func (s *Stmt) MustUpdateModel(model interface{}) sql.Result {
	res, err := s.updateModel(model)
	if err != nil {
		panic(err)
	}
	return res
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
