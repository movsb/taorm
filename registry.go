package taorm

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unsafe"
)

// _FieldInfo stores info about a field in a struct.
type _FieldInfo struct {
	offset uintptr      // the memory offset of the field
	_type  reflect.Type // the reflection type of the field
}

// StructInfo stores info about a struct.
type _StructInfo struct {
	tableName    string                // The database model name for this struct
	fields       map[string]_FieldInfo // struct member info
	fieldstr     string                // fields for inserting
	insertstr    string                // for insert
	updatestr    string                // for update
	insertFields []_FieldInfo          // offsets of member to insert
	pkeyField    _FieldInfo
}

func newStructInfo() *_StructInfo {
	return &_StructInfo{
		fields: make(map[string]_FieldInfo),
	}
}

func (s *_StructInfo) valueOf(out interface{}, field _FieldInfo) reflect.Value {
	return reflect.NewAt(
		field._type,
		unsafe.Pointer(uintptr((*_EmptyEface)(unsafe.Pointer(&out)).ptr)+field.offset),
	).Elem()
}

func (s *_StructInfo) addrOf(out interface{}, field _FieldInfo) interface{} {
	return reflect.NewAt(
		field._type,
		unsafe.Pointer(uintptr((*_EmptyEface)(unsafe.Pointer(&out)).ptr)+field.offset),
	).Interface()
}

func (s *_StructInfo) ptrsOf(out interface{}, fields []string) ([]interface{}, error) {
	ptrs := make([]interface{}, 0, len(fields))
	for _, field := range fields {
		fi, ok := s.fields[field]
		if !ok {
			return nil, &NoPlaceToSaveFieldError{field}
		}
		addr := s.addrOf(out, fi)
		ptrs = append(ptrs, addr)
	}
	return ptrs, nil
}

func (s *_StructInfo) ifacesOf(out interface{}) []interface{} {
	values := make([]interface{}, len(s.insertFields))
	for i, f := range s.insertFields {
		values[i] = reflect.NewAt(
			f._type,
			unsafe.Pointer(uintptr((*_EmptyEface)(unsafe.Pointer(&out)).ptr)+f.offset),
		).Elem().Interface()
	}
	return values
}

func (s *_StructInfo) setPrimaryKey(out interface{}, id int64) {
	pkey := s.valueOf(out, s.pkeyField)
	switch s.pkeyField._type.Kind() {
	case reflect.Uint, reflect.Uint64:
		pkey.SetUint(uint64(id))
	case reflect.Int, reflect.Int64:
		pkey.SetInt(id)
	default:
		panic("cannot set primary key")
	}
}

func (s *_StructInfo) getPrimaryKey(out interface{}) (interface{}, bool) {
	zero := reflect.Zero(s.pkeyField._type).Interface()
	pkv := s.valueOf(out, s.pkeyField).Interface()
	return pkv, pkv != zero
}

// structs maps struct type name to its info.
var structs = make(map[string]*_StructInfo)
var rwLock = &sync.RWMutex{}

// register ...
func register(ty reflect.Type) (*_StructInfo, error) {
	rwLock.Lock()
	defer rwLock.Unlock()

	typeName := structName(ty)
	tableName, err := getTableNameFromType(ty)
	if err != nil {
		return nil, err
	}

	// TODO check name validity
	// TODO name can be empty because of auto-generated table info.

	if si, ok := structs[typeName]; ok {
		return si, nil
	}

	structInfo := newStructInfo()
	structInfo.tableName = tableName
	fieldNames := []string{}

	addStructFields(structInfo, ty, &fieldNames)

	structInfo.fieldstr = strings.Join(fieldNames, ",")
	{
		query := fmt.Sprintf(`INSERT INTO %s `, tableName)
		query += fmt.Sprintf(`(%s) VALUES (%s)`,
			structInfo.fieldstr,
			createSQLInMarks(len(fieldNames)),
		)
		structInfo.insertstr = query
	}
	{
		query := fmt.Sprintf(`UPDATE %s SET `, tableName)
		pairs := []string{}
		for _, name := range fieldNames {
			pairs = append(pairs, name+"=?")
		}
		query += strings.Join(pairs, ",")
		structInfo.updatestr = query
	}
	structs[typeName] = structInfo
	//fmt.Printf("taorm: registered: %s\n", typeName)
	return structInfo, nil
}

func addStructFields(info *_StructInfo, ty reflect.Type, fieldNames *[]string) {
	for i := 0; i < ty.NumField(); i++ {
		f := ty.Field(i)
		if isColumnField(f) {
			columnName := getColumnName(f)
			if columnName == "" {
				continue
			}
			if columnName != "id" {
				*fieldNames = append(*fieldNames, columnName)
			}
			fieldInfo := _FieldInfo{
				offset: f.Offset,
				_type:  f.Type,
			}
			info.fields[columnName] = fieldInfo
			if columnName != "id" {
				info.insertFields = append(info.insertFields, fieldInfo)
			} else {
				info.pkeyField = fieldInfo
			}
		} else if f.Anonymous {
			addStructFields(info, f.Type, fieldNames)
		}
	}
}

// _struct can be any struct-related types.
// e.g.: struct{}, *struct{}, **struct{}, []struct{}, []*struct, []*struct{}, *[]strcut{}, *[]*struct{} ...
func structType(_struct interface{}) (reflect.Type, error) {
	ty := reflect.TypeOf(_struct)
	if ty == nil {
		return nil, &NotStructError{}
	}
	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}
	if ty.Kind() != reflect.Struct {
		return nil, &NotStructError{ty.Kind()}
	}
	return ty, nil
}

var tableNamerType = reflect.TypeOf((*TableNamer)(nil)).Elem()

// getTableName gets the table name for a specific type.
// The type must implement TableNamer.
func getTableNameFromType(ty reflect.Type) (string, error) {
	if ty == nil {
		return ``, &NotStructError{}
	}

	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}

	return getTableNameFromValue(reflect.New(ty).Interface())
}

func getTableNameFromValue(value interface{}) (string, error) {
	if i, ok := value.(TableNamer); ok {
		return i.TableName(), nil
	}
	return ``, nil
}

func getRegistered(_struct interface{}) (*_StructInfo, error) {
	ty, err := structType(_struct)
	if err != nil {
		return nil, err
	}
	name := structName(ty)

	rwLock.RLock()
	if si, ok := structs[name]; ok {
		rwLock.RUnlock()
		return si, nil
	}
	rwLock.RUnlock()
	return register(ty)
}
