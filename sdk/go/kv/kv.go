// Package kv provides access to key value stores within Spin
// components.
package kv

// #include "key-value.h"
import "C"
import (
	"fmt"
	"unsafe"
)

// Store is the Key/Value backend storage.
type Store struct {
	name   string
	active bool
	ptr    C.key_value_store_t
}

// NewStore creates a new instance of Store.
func NewStore(name string) *Store {
	return &Store{name: name}
}

// Open establishes a connection to the key/value storage.
func (s *Store) Open() error {
	return s.open()
}

// Close terminates the connection to Store.
func (s *Store) Close() {
	if s.active {
		C.key_value_close(C.uint32_t(s.ptr))
	}
	s.active = false
}

// Get retrieves a value from Store.
func (s *Store) Get(key string) ([]byte, error) {
	ckey := toCStr(key)
	var ret C.key_value_expected_list_u8_error_t
	C.key_value_get(C.uint32_t(s.ptr), &ckey, &ret)
	if ret.is_err {
		return nil, toErr((*C.key_value_error_t)(unsafe.Pointer(&ret.val)))
	}
	list := (*C.key_value_list_u8_t)(unsafe.Pointer(&ret.val))
	return C.GoBytes(unsafe.Pointer(list.ptr), C.int(list.len)), nil
}

// Delete removes a value from Store.
func (s *Store) Delete(key string) error {
	ckey := toCStr(key)
	var ret C.key_value_expected_unit_error_t
	C.key_value_delete(C.uint32_t(s.ptr), &ckey, &ret)
	if ret.is_err {
		return toErr((*C.key_value_error_t)(unsafe.Pointer(&ret.val)))
	}
	return nil
}

// Set creates a new key/value in Store.
func (s *Store) Set(key string, value []byte) error {
	ckey := toCStr(key)
	cbytes := toCBytes(value)
	var ret C.key_value_expected_unit_error_t
	C.key_value_set(C.uint32_t(s.ptr), &ckey, &cbytes, &ret)
	if ret.is_err {
		return toErr((*C.key_value_error_t)(unsafe.Pointer(&ret.val)))
	}
	return nil
}

// Exists checks if a key exists within Store.
func (s *Store) Exists(key string) (bool, error) {
	ckey := toCStr(key)
	var ret C.key_value_expected_bool_error_t
	C.key_value_exists(C.uint32_t(s.ptr), &ckey, &ret)
	if ret.is_err {
		return false, toErr((*C.key_value_error_t)(unsafe.Pointer(&ret.val)))
	}
	return *(*bool)(unsafe.Pointer(&ret.val)), nil
}

func (s *Store) open() error {
	if s.active {
		return nil
	}
	cname := toCStr(s.name)
	var ret C.key_value_expected_store_error_t
	C.key_value_open(&cname, &ret)
	if ret.is_err {
		return toErr((*C.key_value_error_t)(unsafe.Pointer(&ret.val)))
	}
	s.ptr = *(*C.key_value_store_t)(unsafe.Pointer(&ret.val))
	s.active = true
	return nil
}

func toCBytes(x []byte) C.key_value_list_u8_t {
	return C.key_value_list_u8_t{ptr: (*C.uint8_t)(unsafe.Pointer(&x[0])), len: C.size_t(len(x))}
}

func toCStr(x string) C.key_value_string_t {
	return C.key_value_string_t{ptr: C.CString(x), len: C.size_t(len(x))}
}

func fromCStrList(list *C.key_value_list_string_t) []string {
	var result []string

	listLen := int(list.len)
	slice := unsafe.Slice(list.ptr, listLen)
	for i := 0; i < listLen; i++ {
		str := slice[i]
		result = append(result, C.GoStringN(str.ptr, C.int(str.len)))
	}

	return result
}

// Error types returned from the value store.
const (
	ErrorStoreTableFull = iota
	ErrorNoSuchStore
	ErrorAccessDenied
	ErrorInvalidStore
	ErrorNoSuchKey
	ErrorIO
)

// Error returned from the value store.
type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

func toErr(err *C.key_value_error_t) error {
	switch err.tag {
	case ErrorStoreTableFull:
		return newError(ErrorStoreTableFull, "store table full")

	case ErrorNoSuchStore:
		return newError(ErrorNoSuchStore, "no such store")

	case ErrorAccessDenied:
		return newError(ErrorAccessDenied, "access denied")

	case ErrorInvalidStore:
		return newError(ErrorInvalidStore, "invalid store")

	case ErrorNoSuchKey:
		return newError(ErrorNoSuchKey, "no such key")

	case ErrorIO:
		str := (*C.key_value_string_t)(unsafe.Pointer(&err.val))
		return newError(ErrorIO, fmt.Sprintf("io error: %s", C.GoStringN(str.ptr, C.int(str.len))))

	default:
		return newError(int(err.tag), fmt.Sprintf("unrecognized error: %v", err.tag))
	}
}
