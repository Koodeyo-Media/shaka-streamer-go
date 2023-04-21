package streamer

import (
	"fmt"
	"reflect"
)

/*
A base class for config errors.

Each subclass provides a meaningful, human-readable string representation in
English.
*/
type ConfigError struct {
	ClassName string
	FieldName string
	FieldType string
}

func NewConfigError(classRef interface{}, fieldName string) *ConfigError {
	return &ConfigError{
		FieldName: fieldName,
		ClassName: GetStructName(classRef),
		FieldType: GetStructFieldType(classRef, fieldName),
	}
}

// An error raised when an unrecognized field is encountered in the input.
type UnrecognizedField struct {
	ConfigError
}

func NewUnrecognizedField(classRef interface{}, fieldName string) *UnrecognizedField {
	return &UnrecognizedField{
		ConfigError: *NewConfigError(classRef, fieldName),
	}
}

func (e UnrecognizedField) Error() string {
	return fmt.Sprintf("%s contains unrecognized field: %s", e.ClassName, e.FieldName)
}

// An error raised when a field in the input has the wrong type.
type WrongType struct {
	ConfigError
}

func NewWrongType(classRef interface{}, fieldName string) *WrongType {
	return &WrongType{
		ConfigError: *NewConfigError(classRef, fieldName),
	}
}

func (e WrongType) Error() string {
	return fmt.Sprintf("In %s, %s field requires a %v", e.ClassName, e.FieldName, e.FieldType)
}

// An error raised when a required field is missing from the input.
type MissingRequiredField struct {
	ConfigError
}

func NewMissingRequiredField(classRef interface{}, fieldName string) *MissingRequiredField {
	return &MissingRequiredField{
		ConfigError: *NewConfigError(classRef, fieldName),
	}
}

func (e MissingRequiredField) Error() string {
	return fmt.Sprintf("%s is missing a required field: %s, a %v", e.ClassName, e.FieldName, e.FieldType)
}

// An error raised when a field is malformed.
type MalformedField struct {
	ConfigError
	Reason string
}

func NewMalformedField(classRef interface{}, fieldName, reason string) *MalformedField {
	return &MalformedField{
		ConfigError: *NewConfigError(classRef, fieldName),
		Reason:      reason,
	}
}

func (e MalformedField) Error() string {
	return fmt.Sprintf("In %s, %s field is malformed: %s", e.ClassName, e.FieldName, e.Reason)
}

// An error raised when an input stream is not found.
type InputNotFound struct {
	ClassName string
	TrackNum  int
	MediaType MediaType
	Name      string
}

func NewInputNotFound(i Input) *InputNotFound {
	return &InputNotFound{
		ClassName: reflect.TypeOf(i).Name(),
		TrackNum:  i.TrackNum,
		MediaType: i.MediaType,
		Name:      i.Name,
	}
}

func (e InputNotFound) Error() string {
	return fmt.Sprintf(`In %s, %s track %v was not found in "%s"`, e.ClassName, e.MediaType, e.TrackNum, e.Name)
}

// An error raised when multiple fields are given and only one of them is allowed at a time.
type ConflictingFields struct {
	ClassName  string
	Field1Name string
	Field2Name string
	Field1Type string
	Field2Type string
}

func NewConflictingFields(classRef interface{}, field1Name string, field2Name string) *ConflictingFields {
	return &ConflictingFields{
		Field1Name: field1Name,
		Field2Name: field2Name,
		ClassName:  GetStructName(classRef),
		Field1Type: GetStructFieldType(classRef, field1Name),
		Field2Type: GetStructFieldType(classRef, field2Name),
	}
}

func (e ConflictingFields) Error() string {
	return fmt.Sprintf("In %s, these fields are conflicting: %s a %s and %s a %s\n  consider using only one of them.", e.ClassName, e.Field1Name, e.Field1Type, e.Field2Name, e.Field2Type)
}

type MissingRequiredExclusiveFields struct {
	ConflictingFields
}

func NewMissingRequiredExclusiveFields(classRef interface{}, fieldName1, fieldName2 string) *MissingRequiredExclusiveFields {
	return &MissingRequiredExclusiveFields{
		ConflictingFields: *NewConflictingFields(classRef, fieldName1, fieldName2),
	}
}

func (e MissingRequiredExclusiveFields) Error() string {
	return fmt.Sprintf("%s is missing a required field. Use exactly one of these fields: %s a %s or %s a %s", e.ClassName, e.Field1Name, e.Field1Type, e.Field2Name, e.Field2Type)
}
