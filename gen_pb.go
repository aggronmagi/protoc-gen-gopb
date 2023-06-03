package main

import (
	"fmt"
	"log"
	"strconv"

	"go.uber.org/multierr"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func genEnum(g *protogen.GeneratedFile, f *protogen.File, e *protogen.Enum) (err error) {
	// Enum type declaration.
	leadingComments := appendDeprecationSuffix(e.Comments.Leading,
		e.Desc.ParentFile(),
		e.Desc.Options().(*descriptorpb.EnumOptions).GetDeprecated())
	g.P(leadingComments,
		"type ", e.GoIdent, " int32")

	// Enum value constants.
	g.P("const (")
	for _, value := range e.Values {
		leadingComments := appendDeprecationSuffix(value.Comments.Leading,
			value.Desc.ParentFile(),
			value.Desc.Options().(*descriptorpb.EnumValueOptions).GetDeprecated())
		g.P(leadingComments,
			value.GoIdent, " ", e.GoIdent, " = ", value.Desc.Number(),
			trailingComment(value.Comments.Trailing))
	}
	g.P(")")
	g.P()

	// Enum value maps.
	g.P("// Enum value maps for ", e.GoIdent, ".")
	g.P("var (")
	g.P(e.GoIdent.GoName+"_name", " = map[int32]string{")
	for _, value := range e.Values {
		duplicate := ""
		if value.Desc != e.Desc.Values().ByNumber(value.Desc.Number()) {
			duplicate = "// Duplicate value: "
		}
		g.P(duplicate, value.Desc.Number(), ": ", strconv.Quote(string(value.Desc.Name())), ",")
	}
	g.P("}")
	g.P(e.GoIdent.GoName+"_value", " = map[string]int32{")
	for _, value := range e.Values {
		g.P(strconv.Quote(string(value.Desc.Name())), ": ", value.Desc.Number(), ",")
	}
	g.P("}")
	g.P(")")
	g.P()

	// Enum method.
	//
	// NOTE: A pointer value is needed to represent presence in proto2.
	// Since a proto2 message can reference a proto3 enum, it is useful to
	// always generate this method (even on proto3 enums) to support that case.
	g.P("func (x ", e.GoIdent, ") Enum() *", e.GoIdent, " {")
	g.P("p := new(", e.GoIdent, ")")
	g.P("*p = x")
	g.P("return p")
	g.P("}")
	g.P()

	// String method.
	g.P("func (x ", e.GoIdent, ") String() string {")
	g.P("if name,ok :=", e.GoIdent.GoName+"_name[int32(x)]; ok {")
	g.P("return name")
	g.P("}")
	g.P("return strconv.FormatInt(int64(x), 10)")
	g.P("}")
	g.P()

	g.Import(protogen.GoImportPath("strconv"))
	g.QualifiedGoIdent(protogen.GoIdent{GoName: "Abc", GoImportPath: "strconv"})
	return nil
}

func genMessage(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) (err error) {
	if m.Desc.IsMapEntry() {
		return
	}

	// Message type declaration.
	leadingComments := appendDeprecationSuffix(m.Comments.Leading,
		m.Desc.ParentFile(),
		m.Desc.Options().(*descriptorpb.MessageOptions).GetDeprecated())
	g.P(leadingComments,
		"type ", m.GoIdent, " struct {")
	err = genMessageFields(g, f, m)
	g.P("}")
	g.P()

	if err != nil {
		return err
	}

	genMessageMethods(g, f, m)

	if cfg.zap {
		genZapMessage(g, m)
	}

	for _, e := range m.Enums {
		genEnum(g, f, e)
	}

	for _, m := range m.Messages {
		err = multierr.Append(err, genMessage(g, f, m))
	}
	return
}

func genMessageFields(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) (err error) {
	for _, field := range m.Fields {
		err = multierr.Append(err, genMessageField(g, f, m, field))
	}
	return
}

func genMessageField(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message, field *protogen.Field) (err error) {
	// NOTE: 不支持oneof
	if oneof := field.Oneof; oneof != nil && !oneof.Desc.IsSynthetic() {
		err = fmt.Errorf("%s %s %s is oneof type. not support", f.Desc.FullName(), m.GoIdent, field.GoIdent)
		log.Println(err)
		return
	}
	// weak
	if field.Desc.IsWeak() {
		err = fmt.Errorf("%s %s %s is weak type. not support", f.Desc.FullName(), m.GoIdent, field.GoIdent)
		log.Println(err)
		return
	}
	if field.Desc.HasDefault() {
		err = fmt.Errorf("%s %s %s has default value. not support", f.Desc.FullName(), m.GoIdent, field.GoIdent)
		log.Println(err)
		return
	}
	if field.Desc.Kind() == protoreflect.GroupKind {
		err = fmt.Errorf("%s %s %s is group type. not support", f.Desc.FullName(), m.GoIdent, field.GoIdent)
		log.Println(err)
		return
	}
	goType, pointer := fieldGoType(g, field)
	if pointer {
		goType = "*" + goType
	}
	tags := structTags{
		{"json", fieldJSONTagValue(field)},
		{"db", string(field.Desc.Name())},
	}
	name := field.GoName

	leadingComments := appendDeprecationSuffix(field.Comments.Leading,
		field.Desc.ParentFile(),
		field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())
	g.P(leadingComments,
		name, " ", goType, tags,
		trailingComment(field.Comments.Trailing))
	return
}

func genMessageMethods(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) {
	genMessageBaseMethods(g, f, m)
	if cfg.get {
		genMessageGetterMethods(g, f, m)
	}
	genMessageMarshal(g, f, m)
	genMessageUnmarshal(g, f, m)
	genMessageSize(g, f, m)
}

func genMessageBaseMethods(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) {
	// Reset method.
	g.P("func (x *", m.GoIdent, ") Reset() {")
	g.P("*x = ", m.GoIdent, "{}")
	g.P("}")
	g.P()

	// // String method.
	// g.P("func (x *", m.GoIdent, ") String() string {")
	// g.P("return ", protoimplPackage.Ident("X"), ".MessageStringOf(x)")
	// g.P("}")
	// g.P()
}

func genMessageGetterMethods(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) {
	for _, field := range m.Fields {
		// Getter for parent oneof.
		if oneof := field.Oneof; oneof != nil && oneof.Fields[0] == field && !oneof.Desc.IsSynthetic() {
			// not supoort
		}

		// Getter for message field.
		goType, pointer := fieldGoType(g, field)
		defaultValue := fieldDefaultValue(g, f, m, field)
		leadingComments := appendDeprecationSuffix("",
			field.Desc.ParentFile(),
			field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated())

		g.P(leadingComments, "func (x *", m.GoIdent, ") Get", field.GoName, "() ", goType, " {")
		if !field.Desc.HasPresence() || defaultValue == "nil" {
			g.P("if x != nil {")
		} else {
			g.P("if x != nil && x.", field.GoName, " != nil {")
		}
		star := ""
		if pointer {
			star = "*"
		}
		g.P("return ", star, " x.", field.GoName)
		g.P("}")
		g.P("return ", defaultValue)
		g.P("}")
		g.P()
	}
}

// fieldGoType returns the Go type used for a field.
//
// If it returns pointer=true, the struct field is a pointer to the type.
func fieldGoType(g *protogen.GeneratedFile, field *protogen.Field) (goType string, pointer bool) {
	if field.Desc.IsWeak() {
		return "struct{}", false
	}

	pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		goType = "bool"
	case protoreflect.EnumKind:
		goType = g.QualifiedGoIdent(field.Enum.GoIdent)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		goType = "int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		goType = "uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		goType = "int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		goType = "uint64"
	case protoreflect.FloatKind:
		goType = "float32"
	case protoreflect.DoubleKind:
		goType = "float64"
	case protoreflect.StringKind:
		goType = "string"
	case protoreflect.BytesKind:
		goType = "[]byte"
		pointer = false // rely on nullability of slices for presence
	case protoreflect.MessageKind, protoreflect.GroupKind:
		goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		pointer = false // pointer captured as part of the type
	}
	switch {
	case field.Desc.IsList():
		return "[]" + goType, false
	case field.Desc.IsMap():
		keyType, _ := fieldGoType(g, field.Message.Fields[0])
		valType, _ := fieldGoType(g, field.Message.Fields[1])
		return fmt.Sprintf("map[%v]%v", keyType, valType), false
	}
	return goType, pointer
}

func fieldDefaultValue(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message, field *protogen.Field) string {
	if field.Desc.IsList() {
		return "nil"
	}
	// if field.Desc.HasDefault() {
	// 	defVarName := "Default_" + m.GoIdent.GoName + "_" + field.GoName
	// 	if field.Desc.Kind() == protoreflect.BytesKind {
	// 		return "append([]byte(nil), " + defVarName + "...)"
	// 	}
	// 	return defVarName
	// }
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		return "false"
	case protoreflect.StringKind:
		return `""`
	case protoreflect.MessageKind, protoreflect.GroupKind, protoreflect.BytesKind:
		return "nil"
	case protoreflect.EnumKind:
		val := field.Enum.Values[0]
		if val.GoIdent.GoImportPath == f.GoImportPath {
			return g.QualifiedGoIdent(val.GoIdent)
		} else {
			// If the enum value is declared in a different Go package,
			// reference it by number since the name may not be correct.
			// See https://github.com/golang/protobuf/issues/513.
			return g.QualifiedGoIdent(field.Enum.GoIdent) + "(" + strconv.FormatInt(int64(val.Desc.Number()), 10) + ")"
		}
	default:
		return "0"
	}
}
