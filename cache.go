package gopoet_protogen

import (
	"fmt"
	"github.com/jhump/gopoet"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"sync"
)

type (
	// Cache implements a type cache, that may be populated by feeding it protogen.File values, see also AddFile.
	Cache struct {
		once sync.Once
		data map[protoreflect.FullName]protogen.GoIdent
	}
)

var (
	bytesType = gopoet.SliceType(gopoet.ByteType)
)

// AddFile loads the given file into the cache, note that it is not safe to call concurrently.
// It is recommended that all files (provided by protogen.Plugin) are loaded into the cache, prior to any generation
// activities that might use it.
func (x *Cache) AddFile(v *protogen.File) {
	x.once.Do(x.init)
	for _, v := range v.Enums {
		x.addEnum(v)
	}
	for _, v := range v.Messages {
		x.addMessage(v)
	}
}

// MessageType retrieves the gopoet type name for a given message from the cache, note that the type must be loaded
// into the cache (by using AddFile on the parent file) beforehand, otherwise it will panic.
func (x *Cache) MessageType(v protoreflect.MessageDescriptor) gopoet.TypeName {
	x.once.Do(x.init)
	if v != nil {
		if v := x.lookup(v.FullName()); v != nil {
			return v
		}
	}
	panic(fmt.Sprintf("unknown type: %v", v))
}

// MessageFields returns information for all the golang fields generated for a given message, where all fields must
// exist in the cache. Oneof fields are represented by a single value.
func (x *Cache) MessageFields(v *protogen.Message) []Field {
	x.once.Do(x.init)
	var (
		fields []Field
		seen   = make(map[string]*goField)
	)
	for _, field := range v.Fields {
		var name string
		if field.Oneof != nil {
			name = field.Oneof.GoName
		} else {
			name = field.GoName
		}
		v := seen[name]
		if v == nil {
			v = &goField{cache: x, name: name, oneOf: field.Oneof}
			fields = append(fields, v)
			seen[name] = v
		}
		if v.oneOf != field.Oneof {
			panic(field)
		}
		v.fields = append(v.fields, field)
	}
	return fields
}

func (x *Cache) init() {
	x.data = make(map[protoreflect.FullName]protogen.GoIdent)
}

func (x *Cache) addEnum(v *protogen.Enum) {
	x.once.Do(x.init)
	x.data[v.Desc.FullName()] = v.GoIdent
	for _, v := range v.Values {
		x.data[v.Desc.FullName()] = v.GoIdent
	}
}

func (x *Cache) addMessage(v *protogen.Message) {
	x.once.Do(x.init)
	x.data[v.Desc.FullName()] = v.GoIdent
	for _, v := range v.Enums {
		x.addEnum(v)
	}
	for _, v := range v.Messages {
		x.addMessage(v)
	}
}

func (x *Cache) enumType(v protoreflect.EnumDescriptor) gopoet.TypeName {
	x.once.Do(x.init)
	if v != nil {
		if v := x.lookup(v.FullName()); v != nil {
			return v
		}
	}
	panic(fmt.Sprintf("unknown type: %v", v))
}

func (x *Cache) lookup(fullName protoreflect.FullName) gopoet.TypeName {
	if ident := x.data[fullName]; ident != (protogen.GoIdent{}) {
		return gopoet.NamedType(gopoet.NewPackage(string(ident.GoImportPath)).Symbol(ident.GoName))
	}
	return nil
}

func (x *Cache) fieldType(v protoreflect.FieldDescriptor) (t gopoet.TypeName) {
	// https://github.com/jhump/goprotoc/blob/70c8197ef4ea66d11022326b63050f6fa10f6b29/plugins/names.go#L337
	x.once.Do(x.init)
	if v.IsMap() {
		return gopoet.MapType(x.fieldType(v.MapKey()), x.fieldType(v.MapValue()))
	}
	switch descriptorpb.FieldDescriptorProto_Type(v.Kind()) {
	case descriptorpb.FieldDescriptorProto_TYPE_BOOL:
		t = gopoet.BoolType
	case descriptorpb.FieldDescriptorProto_TYPE_STRING:
		t = gopoet.StringType
	case descriptorpb.FieldDescriptorProto_TYPE_BYTES:
		t = bytesType
	case descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32:
		t = gopoet.Int32Type
	case descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64:
		t = gopoet.Int64Type
	case descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32:
		t = gopoet.Uint32Type
	case descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64:
		t = gopoet.Uint64Type
	case descriptorpb.FieldDescriptorProto_TYPE_FLOAT:
		t = gopoet.Float32Type
	case descriptorpb.FieldDescriptorProto_TYPE_DOUBLE:
		t = gopoet.Float64Type
	case descriptorpb.FieldDescriptorProto_TYPE_GROUP,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		t = gopoet.PointerType(x.MessageType(v.Message()))
	case descriptorpb.FieldDescriptorProto_TYPE_ENUM:
		t = x.enumType(v.Enum())
	default:
		panic(fmt.Sprintf("unknown type: %v", v))
	}
	if v.IsList() {
		t = gopoet.SliceType(t)
	}
	if v.ParentFile().Syntax() != protoreflect.Proto3 && t.Kind() != gopoet.KindPtr && t.Kind() != gopoet.KindSlice {
		// for proto2, type is pointer or slice
		t = gopoet.PointerType(t)
	}
	return
}
