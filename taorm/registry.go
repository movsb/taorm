package taorm

import (
	"fmt"
	"reflect"
	"sync"
)

// _FieldInfo stores info about a field in a struct.
type _FieldInfo struct {
	offset uintptr      // the memory offset of the field
	kind   reflect.Kind // the reflection kind of the field
}

// StructInfo stores info about a struct.
type _StructInfo struct {
	tableName string                // The database model name for this struct
	fields    map[string]_FieldInfo // struct member info
}

func newStructInfo() *_StructInfo {
	return &_StructInfo{
		fields: make(map[string]_FieldInfo),
	}
}

// FieldPointers returns fields' pointers as interface slice.
//
// base: specifies the base address of a struct.
func (s *_StructInfo) FieldPointers(base uintptr, fields []string) ([]interface{}, error) {
	ptrs := make([]interface{}, 0, len(fields))
	for _, field := range fields {
		fi, ok := s.fields[field]
		if !ok {
			return nil, &NoPlaceToSaveFieldError{field}
		}
		i := ptrToInterface(base+fi.offset, fi.kind)
		if i == nil {
			return nil, &UnknownFieldKindError{field, fi.kind}
		}
		ptrs = append(ptrs, i)
	}
	return ptrs, nil
}

// structs maps struct type name to its info.
var structs = make(map[string]*_StructInfo)
var rwLock = &sync.RWMutex{}

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
	for i := 0; i < ty.NumField(); i++ {
		f := ty.Field(i)
		if isColumnField(f) {
			columnName := getColumnName(f)
			if columnName == "" {
				continue
			}
			fieldInfo := _FieldInfo{
				offset: f.Offset,
				kind:   f.Type.Kind(),
			}
			structInfo.fields[columnName] = fieldInfo
		}
	}
	structs[typeName] = structInfo
	fmt.Printf("taorm: registered: %s\n", typeName)
	return structInfo, nil
}

func getRegistered(_struct interface{}) (*_StructInfo, error) {
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

	name := structName(ty)

	rwLock.RLock()
	if si, ok := structs[name]; ok {
		rwLock.RUnlock()
		return si, nil
	}

	rwLock.RUnlock()
	return register(ty, "")
}

func getPointers(out interface{}, columns []string) ([]interface{}, error) {
	base := baseFromInterface(out)
	info, err := getRegistered(out)
	if err != nil {
		return nil, err
	}
	return info.FieldPointers(base, columns)
}
