package genparse

import (
	"fmt"
	"log"
	"strings"

	"github.com/aggronmagi/protoc-gen-gopb/gengo"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// 外部配置
var (
	Getter  bool
	WirePkg string = "google.golang.org/protobuf/encoding/protowire"
	Zap     bool   = true
)

// 版本信息
var (
	Version = "0.0.3"
)

func ParseEnum(t *gengo.GenerateStruct, g *protogen.GeneratedFile, f *protogen.File, e *protogen.Enum) (err error) {
	enum := &gengo.GenerateEnums{}
	enum.LeadingComments = appendDeprecationSuffix(e.Comments.Leading,
		e.Desc.ParentFile(),
		e.Desc.Options().(*descriptorpb.EnumOptions).GetDeprecated()).String()
	enum.TypeName = g.QualifiedGoIdent(e.GoIdent)
	enum.GoName = e.GoIdent.GoName

	for _, value := range e.Values {
		val := &gengo.GenerateEnumValue{}
		val.LeadingComments = appendDeprecationSuffix(value.Comments.Leading,
			value.Desc.ParentFile(),
			value.Desc.Options().(*descriptorpb.EnumValueOptions).GetDeprecated()).String()
		val.TrailingComment = trailingComment(value.Comments.Trailing).String()
		val.Desc = string(value.Desc.Name())
		if value.Desc != e.Desc.Values().ByNumber(value.Desc.Number()) {
			val.Duplicate = "// Duplicate value: "
		}
		val.Num = int32(value.Desc.Number())
		val.ValueName = g.QualifiedGoIdent(value.GoIdent)
		enum.Values = append(enum.Values, val)
	}

	t.Enums = append(t.Enums, enum)

	g.Import(protogen.GoImportPath("strconv"))
	g.QualifiedGoIdent(protogen.GoIdent{GoName: "Abc", GoImportPath: "strconv"})
	return
}

func ParseMessage(t *gengo.GenerateStruct, g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) (err error) {
	if m.Desc.IsMapEntry() {
		return
	}

	msg := &gengo.GenerateMessage{}
	msg.LeadingComments = appendDeprecationSuffix(m.Comments.Leading,
		m.Desc.ParentFile(),
		m.Desc.Options().(*descriptorpb.MessageOptions).GetDeprecated()).String()
	msg.TypeName = g.QualifiedGoIdent(m.GoIdent)
	msg.GoName = m.GoIdent.GoName
	msg.GenGetter = Getter

	for _, field := range m.Fields {
		gf, ne := parseMessageField(msg, g, f, m, field)
		err = multierr.Append(err, ne)
		if gf != nil {
			msg.Fields = append(msg.Fields, gf)
		}
	}
	t.Messages = append(t.Messages, msg)

	if Zap {
		msg.CustomTemplates = append(msg.CustomTemplates, "genzap")
	}

	// sub enum
	for _, en := range m.Enums {
		ParseEnum(t, g, f, en)
	}
	// sub message
	for _, msg := range m.Messages {
		ParseMessage(t, g, f, msg)
	}
	return
}

func parseMessageField(msg *gengo.GenerateMessage, g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message, field *protogen.Field) (genField *gengo.GenerateField, err error) {
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

	genField = &gengo.GenerateField{}

	// 类型相关
	genField.LeadingComments = appendDeprecationSuffix(field.Comments.Leading,
		field.Desc.ParentFile(),
		field.Desc.Options().(*descriptorpb.FieldOptions).GetDeprecated()).String()
	genField.TrailingComment = trailingComment(field.Comments.Trailing).String()
	genField.TypeName = goType
	genField.GoName = field.GoName
	if goType == "[]byte" {
		genField.GoType = goType
	} else {
		//genField.GoType = strings.TrimPrefix(goType, "[]")
		genField.GoType = strings.TrimPrefix(strings.TrimPrefix(goType, "[]"), "*")
	}
	genField.Tip = msg.GoName + "." + field.GoName
	// getter 相关
	defaultValue := fieldDefaultValue(g, f, m, field)
	genField.GetNilCheck = !field.Desc.HasPresence() || defaultValue == "nil"
	genField.DefaultValue = defaultValue
	// tag 相关
	genField.DescNum = int(field.Desc.Number())
	genField.DescName = string(field.Desc.Name())
	genField.DescType, genField.WireType = switchProtoType(field.Desc.Kind())
	genField.IsList = field.Desc.IsList()
	genField.IsMap = field.Desc.IsMap()
	genField.Kind = field.Desc.Kind()

	// 序列化
	switch {
	case field.Desc.IsMap():
		genField.CheckNotEmpty = func(vname string) string {
			return "len(" + vname + ") > 0"
		}

		genField.MapKey, err = parseMessageField(msg, g, f, m, field.Message.Fields[0])
		if err != nil {
			return
		}

		genField.MapValue, err = parseMessageField(msg, g, f, m, field.Message.Fields[1])
		if err != nil {
			return
		}
		genField.TemplateDecode = "decode.map"
		genField.TemplateSize = "size.map"
		genField.TemplateEncode = "encode.map"

	case field.Desc.IsList():
		genField.CheckNotEmpty = func(vname string) string {
			return "len(" + vname + ") > 0"
		}

		if field.Desc.IsPacked() {
			parseFillListPackedFiled(g, genField, field)
		} else {
			parseFillListNoPackedFiled(g, genField, field)
		}

	default:
		parseFillBasicFiled(g, genField, field)
	}

	// import
	g.Import(protogen.GoImportPath(WirePkg))
	g.QualifiedGoIdent(protogen.GoIdent{GoName: "VarintType", GoImportPath: protogen.GoImportPath(WirePkg)})
	g.Import(protogen.GoImportPath("errors"))
	g.QualifiedGoIdent(protogen.GoIdent{GoName: "New", GoImportPath: "errors"})

	return
}

func switchProtoType(kind protoreflect.Kind) (typ int, desc string) {
	switch kind {
	case protoreflect.BoolKind, protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind,
		protoreflect.Int64Kind, protoreflect.Uint64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		desc = "protowire.VarintType"
		typ = int(protowire.VarintType)
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind:
		desc = "protowire.Fixed32Type"
		typ = int(protowire.Fixed32Type)
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		desc = "protowire.Fixed64Type"
		typ = int(protowire.Fixed64Type)
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.MessageKind:
		desc = "protowire.BytesType"
		typ = int(protowire.BytesType)
	}
	return
}

func parseFillBasicFiled(g *protogen.GeneratedFile, genField *gengo.GenerateField, field *protogen.Field) {
	genField.CheckNotEmpty = func(x string) string {
		return x + " != 0"
	}
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		genField.CheckNotEmpty = func(x string) string {
			return x
		}
		genField.TemplateSize = "size.bool"
		genField.TemplateEncode = "encode.bool"
		genField.TemplateDecode = "decode.bool"

	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		genField.TemplateSize = "size.varint"
		genField.TemplateEncode = "encode.varint"
		genField.TemplateDecode = "decode.varint"

	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		genField.TemplateSize = "size.sint"
		genField.TemplateEncode = "encode.sint"
		genField.TemplateDecode = "decode.sint"

	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		genField.TemplateSize = "size.fix32"
		genField.TemplateEncode = "encode.fix32"
		genField.TemplateDecode = "decode.fix32"

	case protoreflect.FloatKind:
		genField.TemplateSize = "size.float"
		genField.TemplateEncode = "encode.float"
		genField.TemplateDecode = "decode.float"

		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		genField.TemplateSize = "size.fix64"
		genField.TemplateEncode = "encode.fix64"
		genField.TemplateDecode = "decode.fix64"

	case protoreflect.DoubleKind:
		genField.TemplateSize = "size.double"
		genField.TemplateEncode = "encode.double"
		genField.TemplateDecode = "decode.double"

		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float64bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		genField.CheckNotEmpty = func(x string) string {
			return "len(" + x + ") > 0"
		}
		genField.TemplateSize = "size.string"
		genField.TemplateEncode = "encode.string"
		genField.TemplateDecode = "decode.string"

	case protoreflect.BytesKind:
		genField.CheckNotEmpty = func(x string) string {
			return "len(" + x + ") > 0"
		}
		genField.TemplateSize = "size.bytes"
		genField.TemplateEncode = "encode.bytes"
		genField.TemplateDecode = "decode.bytes"

	case protoreflect.MessageKind:
		genField.CheckNotEmpty = func(x string) string {
			return x + " != nil"
		}
		genField.TemplateSize = "size.message"
		genField.TemplateEncode = "encode.message"
		genField.TemplateDecode = "decode.message"

	}
}

func parseFillListPackedFiled(g *protogen.GeneratedFile, genField *gengo.GenerateField, field *protogen.Field) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		genField.TemplateSize = "size.packed.bool"
		genField.TemplateEncode = "encode.packed.bool"
		genField.TemplateDecode = "decode.slice.bool"

	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		genField.TemplateSize = "size.packed.varint"
		genField.TemplateEncode = "encode.packed.varint"
		genField.TemplateDecode = "decode.slice.varint"

	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		genField.TemplateSize = "size.packed.sint"
		genField.TemplateEncode = "encode.packed.sint"
		genField.TemplateDecode = "decode.slice.sint"

	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		genField.TemplateSize = "size.packed.fix32"
		genField.TemplateEncode = "encode.packed.fix32"
		genField.TemplateDecode = "decode.slice.fix32"

	case protoreflect.FloatKind:
		genField.TemplateSize = "size.packed.float"
		genField.TemplateEncode = "encode.packed.float"
		genField.TemplateDecode = "decode.slice.float"

		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		genField.TemplateSize = "size.packed.fix64"
		genField.TemplateEncode = "encode.packed.fix64"
		genField.TemplateDecode = "decode.slice.fix64"

	case protoreflect.DoubleKind:
		genField.TemplateSize = "size.packed.double"
		genField.TemplateEncode = "encode.packed.double"
		genField.TemplateDecode = "decode.slice.double"

		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float64bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		genField.CheckNotEmpty = func(x string) string {
			return "len(" + x + ") > 0"
		}
		genField.TemplateSize = "size.packed.string"
		genField.TemplateEncode = "encode.packed.string"
		genField.TemplateDecode = "decode.slice.string"

	case protoreflect.BytesKind:
		genField.CheckNotEmpty = func(x string) string {
			return "len(" + x + ") > 0"
		}
		genField.TemplateSize = "size.packed.bytes"
		genField.TemplateEncode = "encode.packed.bytes"
		genField.TemplateDecode = "decode.slice.bytes"

	case protoreflect.MessageKind:
		genField.CheckNotEmpty = func(x string) string {
			return x + " != nil"
		}
		genField.TemplateSize = "size.packed.message"
		genField.TemplateEncode = "encode.packed.message"
		genField.TemplateDecode = "decode.slice.message"

	}
}

func parseFillListNoPackedFiled(g *protogen.GeneratedFile, genField *gengo.GenerateField, field *protogen.Field) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		genField.TemplateSize = "size.nopack.bool"
		genField.TemplateEncode = "encode.nopack.bool"
		genField.TemplateDecode = "decode.slice.bool"

	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		genField.TemplateSize = "size.nopack.varint"
		genField.TemplateEncode = "encode.nopack.varint"
		genField.TemplateDecode = "decode.slice.varint"

	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		genField.TemplateSize = "size.nopack.sint"
		genField.TemplateEncode = "encode.nopack.sint"
		genField.TemplateDecode = "decode.slice.sint"

	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		genField.TemplateSize = "size.nopack.fix32"
		genField.TemplateEncode = "encode.nopack.fix32"
		genField.TemplateDecode = "decode.slice.fix32"

	case protoreflect.FloatKind:
		genField.TemplateSize = "size.nopack.float"
		genField.TemplateEncode = "encode.nopack.float"
		genField.TemplateDecode = "decode.slice.float"

		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		genField.TemplateSize = "size.nopack.fix64"
		genField.TemplateEncode = "encode.nopack.fix64"
		genField.TemplateDecode = "decode.slice.fix64"

	case protoreflect.DoubleKind:
		genField.TemplateSize = "size.nopack.double"
		genField.TemplateEncode = "encode.nopack.double"
		genField.TemplateDecode = "decode.slice.double"

		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float64bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		genField.CheckNotEmpty = func(x string) string {
			return "len(" + x + ") > 0"
		}
		genField.TemplateSize = "size.nopack.string"
		genField.TemplateEncode = "encode.nopack.string"
		genField.TemplateDecode = "decode.slice.string"

	case protoreflect.BytesKind:
		genField.CheckNotEmpty = func(x string) string {
			return "len(" + x + ") > 0"
		}
		genField.TemplateSize = "size.nopack.bytes"
		genField.TemplateEncode = "encode.nopack.bytes"
		genField.TemplateDecode = "decode.slice.bytes"

	case protoreflect.MessageKind:
		genField.CheckNotEmpty = func(x string) string {
			return x + " != nil"
		}
		genField.TemplateSize = "size.nopack.message"
		genField.TemplateEncode = "encode.nopack.message"
		genField.TemplateDecode = "decode.slice.message"

	}
}
