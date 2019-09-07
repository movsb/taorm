package taorm

import (
	"database/sql"
	"reflect"
	"unsafe"
)

// ScanRows scans result rows into out.
//
// out can be either *primitive, *Struct, *[]Struct, or *[]*Struct.
func ScanRows(out interface{}, tx _SQLCommon, query string, args ...interface{}) (_err error) {
	defer func() { _err = WrapError(_err) }()

	rows, err := tx.Query(query, args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	ty := reflect.TypeOf(out)
	if ty.Kind() != reflect.Ptr {
		return ErrInvalidOut
	}

	ty = ty.Elem()
	switch ty.Kind() {
	case reflect.Struct:
		info, err := getRegistered(out)
		if err != nil {
			return err
		}
		if rows.Next() {
			pointers, err := info.ptrsOf(out, columns)
			if err != nil {
				return err
			}
			return rows.Scan(pointers...)
		}
		err = rows.Err()
		if err == nil {
			err = sql.ErrNoRows
		}
		return err
	case reflect.Slice:
		slice := reflect.MakeSlice(ty, 0, 0)
		ty = ty.Elem()
		isPtr := ty.Kind() == reflect.Ptr
		if isPtr {
			ty = ty.Elem()
		}
		if ty.Kind() != reflect.Struct {
			return ErrInvalidOut
		}
		info, err := getRegistered(reflect.NewAt(ty, unsafe.Pointer(nil)).Interface())
		if err != nil {
			return err
		}
		if isPtr {
			for rows.Next() {
				elem := reflect.New(ty)
				elemPtr := elem.Interface()
				pointers, err := info.ptrsOf(elemPtr, columns)
				if err != nil {
					return err
				}
				if err := rows.Scan(pointers...); err != nil {
					return err
				}
				slice = reflect.Append(slice, elem)
			}
		} else {
			elem := reflect.New(ty)
			elemPtr := elem.Interface()
			pointers, err := info.ptrsOf(elemPtr, columns)
			if err != nil {
				return err
			}
			for rows.Next() {
				if err := rows.Scan(pointers...); err != nil {
					return err
				}
				slice = reflect.Append(slice, elem.Elem())
			}
		}
		reflect.ValueOf(out).Elem().Set(slice)
		return rows.Err()
	default:
		if len(columns) != 1 {
			return ErrInvalidOut
		}
		if rows.Next() {
			return rows.Scan(out)
		}
		err = rows.Err()
		if err == nil {
			err = sql.ErrNoRows
		}
		return err
	}
}

// MustScanRows ...
func MustScanRows(out interface{}, tx _SQLCommon, query string, args ...interface{}) {
	if err := ScanRows(out, tx, query, args...); err != nil {
		panic(err)
	}
}
