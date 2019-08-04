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

func (s *_StructInfo) baseOf(out interface{}) uintptr {
	return uintptr((*_EmptyEface)(unsafe.Pointer(&out)).ptr)
}

func (s *_StructInfo) valueOf(out interface{}, field _FieldInfo) reflect.Value {
	addr := unsafe.Pointer(s.baseOf(out) + field.offset)
	return reflect.NewAt(field._type, addr).Elem()
}

func (s *_StructInfo) addrOf(out interface{}, field _FieldInfo) interface{} {
	addr := unsafe.Pointer(s.baseOf(out) + field.offset)
	return reflect.NewAt(field._type, addr).Interface()
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
	base := s.baseOf(out)
	for i, f := range s.insertFields {
		addr := unsafe.Pointer(base + f.offset)
		values[i] = reflect.NewAt(f._type, addr).Elem().Interface()
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

func (s *_StructInfo) getPrimaryKey(out interface{}) interface{} {
	return s.valueOf(out, s.pkeyField).Interface()
}

// structs maps struct type name to its info.
var structs = make(map[string]*_StructInfo)
var rwLock = &sync.RWMutex{}

// Register ...
func Register(_struct interface{}, tableName string) error {
	_, err := register(reflect.TypeOf(_struct), tableName)
	return WrapError(err)
}

// register ...
func register(ty reflect.Type, tableName string) (*_StructInfo, error) {
	rwLock.Lock()
	defer rwLock.Unlock()

	typeName := structName(ty)

	if si, ok := structs[typeName]; ok {
		return si, nil
	}

	structInfo := newStructInfo()
	structInfo.tableName = tableName
	fieldNames := []string{}
	for i := 0; i < ty.NumField(); i++ {
		f := ty.Field(i)
		if isColumnField(f) {
			columnName := getColumnName(f)
			if columnName == "" {
				continue
			}
			if columnName != "id" {
				fieldNames = append(fieldNames, columnName)
			}
			fieldInfo := _FieldInfo{
				offset: f.Offset,
				_type:  f.Type,
			}
			structInfo.fields[columnName] = fieldInfo
			if columnName != "id" {
				structInfo.insertFields = append(structInfo.insertFields, fieldInfo)
			} else {
				structInfo.pkeyField = fieldInfo
			}
		}
	}
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

func structType(_struct interface{}) (reflect.Type, error) {
	ty := reflect.TypeOf(_struct)
	if ty == nil {
		return nil, &NotStructError{}
	}
	if ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	if ty.Kind() != reflect.Struct {
		return nil, &NotStructError{ty.Kind()}
	}
	return ty, nil
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
	return nil, fmt.Errorf("not registered:" + name)
}
