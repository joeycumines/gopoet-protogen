package gopoet_protogen

import (
	"github.com/jhump/gopoet"
	"google.golang.org/protobuf/compiler/protogen"
	"sync"
)

type (
	// Field models the protobuf fields for a message, with oneof fields merged into a single value.
	Field interface {
		// Name is the name of the field.
		Name() string
		// OneOf will be non-nil for oneof fields.
		// Note that fields with the optional field rule will be represented as oneof fields.
		// Check the descriptor's IsSynthetic() method to handle that case, see also FieldIsOptional.
		OneOf() *protogen.Oneof
		// Fields are all the input protogen.Field values for this Field, typically there will be one, but there may
		// be more than one in the case of oneof fields.
		Fields() []*protogen.Field
		// Type returns the gopoet.TypeName for this field, which will be the unexported interface type in the case of
		// oneof fields (it's the return value of the getter method, in all cases).
		Type() gopoet.TypeName
		// Getter returns the gopoet.MethodType for the generated getter method (the generic one, for oneof fields).
		Getter() gopoet.MethodType
		// OneOfFields returns the same information as Type and Getter and Fields, for each of the actual oneof fields,
		// if any.
		OneOfFields() []OneOfField
	}

	// OneOfField models the actual type information for a specific oneof field.
	OneOfField struct {
		// Field is the input protogen.Field for this OneOfField.
		Field *protogen.Field
		// Type returns the gopoet.TypeName for the actual golang field.
		Type gopoet.TypeName
		// Getter is the gopoet.MethodType for the generated getter method (it's return type is Type).
		Getter gopoet.MethodType
	}

	goField struct {
		cache       *Cache
		name        string
		oneOf       *protogen.Oneof
		fields      []*protogen.Field
		once        sync.Once
		typeName    gopoet.TypeName
		getter      gopoet.MethodType
		oneOfFields []OneOfField
	}
)

var (
	_ Field = (*goField)(nil)
)

// FieldIsOptional returns true if the field is optional.
func FieldIsOptional(field Field) bool {
	if oneOf := field.OneOf(); oneOf != nil && oneOf.Desc.IsSynthetic() {
		return true
	}
	return false
}

func (x *goField) Name() string { return x.name }

func (x *goField) OneOf() *protogen.Oneof { return x.oneOf }

func (x *goField) Fields() []*protogen.Field { return x.fields }

func (x *goField) Type() gopoet.TypeName {
	x.once.Do(x.init)
	return x.typeName
}

func (x *goField) Getter() gopoet.MethodType {
	x.once.Do(x.init)
	return x.getter
}

func (x *goField) OneOfFields() []OneOfField {
	x.once.Do(x.init)
	return x.oneOfFields
}

func (x *goField) init() {
	if x.oneOf != nil && !x.oneOf.Desc.IsSynthetic() {
		// https://github.com/protocolbuffers/protobuf-go/blob/fc9592f7ac4bade8f83e636263f8f07715c698d1/cmd/protoc-gen-go/internal_gengo/main.go#L810
		x.typeName = gopoet.NamedType(gopoet.NewPackage(string(x.oneOf.GoIdent.GoImportPath)).Symbol("is" + x.oneOf.GoIdent.GoName))
		for _, field := range x.fields {
			x.oneOfFields = append(x.oneOfFields, OneOfField{
				Field:  field,
				Type:   gopoet.NamedType(gopoet.NewPackage(string(field.GoIdent.GoImportPath)).Symbol(field.GoIdent.GoName)),
				Getter: gopoet.MethodType{Name: `Get` + field.GoName, Signature: gopoet.Signature{Results: []gopoet.ArgType{{Type: x.cache.fieldType(field.Desc)}}}},
			})
		}
	} else {
		x.typeName = x.cache.fieldType(x.fields[0].Desc)
	}
	x.getter = gopoet.MethodType{Name: `Get` + x.name, Signature: gopoet.Signature{Results: []gopoet.ArgType{{Type: x.typeName}}}}
}
