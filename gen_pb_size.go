package main

import (
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func genMessageSize(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) {
	g.P("// MarshalSize calc marshal data need space")
	g.P("func (x *", m.GoIdent, ") MarshalSize()(size int) {")
	for _, field := range m.Fields {
		genSizeMarshalField(g, field)
	}
	g.P("return")
	g.P("}")
	g.P()
}

func genSizeMarshalField(g *protogen.GeneratedFile, field *protogen.Field) {
	// 数组
	if field.Desc.IsList() {
		genSizeListField(g, field)
		return
	}
	// map
	if field.Desc.IsMap() {
		genSizeMapField(g, field)
		return
	}
	// 序列化基础数据类型
	genSizeBasicField(g, field, "x."+field.GoName)
}

func genSizeBasicField(g *protogen.GeneratedFile, field *protogen.Field, vname string) {
	//pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("if ", vname, "{")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		g.P("if ", vname, " != 0 {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		g.P("if ", vname, " != 0 {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		g.P("if ", vname, " != 0 {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.FloatKind:
		g.P("if ", vname, " != 0 {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		g.P("if ", vname, " != 0 {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.DoubleKind:
		g.P("if ", vname, " != 0 {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		g.P("if len(", vname, ") > 0{")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.BytesKind:
		g.P("if len(", vname, ") > 0{")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.MessageKind:
		//goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		//pointer = false // pointer captured as part of the type

		g.P("if ", vname, " != nil {")
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size")
		g.P("}")
	case protoreflect.GroupKind:
		// unsupport
	}
	return //  goType, pointer
}

func genSizeListField(g *protogen.GeneratedFile, field *protogen.Field) {

	g.P("if len(x.", field.GoName, ") > 0{")
	defer g.P("}")

	// packed=false ,每个元素单独发送
	if !field.Desc.IsPacked() {
		g.P("for _,item := range x.", field.GoName, "{")
		genSizeBasicField(g, field, "item")
		g.P("}")
		return
	}

	// packed=true 元素一起发送
	g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("size += len(x.", field.GoName, ")")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind:
		g.P("size += len(x.", field.GoName, ")*4")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		g.P("size += len(x.", field.GoName, ")*8")
	default:
		g.P("for _,item := range x.", field.GoName, "{")
		genFieldSizeFunc(g, field, "item", "size")
		g.P("}")
	}
	return //  goType, pointer
}

func genSizeMapField(g *protogen.GeneratedFile, field *protogen.Field) {
	g.P("if len(x.", field.GoName, ") > 0{")
	defer g.P("}")

	g.P("for mk,mv := range x.", field.GoName, " {")
	g.P("_ = mk")
	g.P("_ = mv")
	defer g.P("}")

	// tag
	g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")

	// 长度计
	g.P("size += ", protowire.SizeTag(1), "+", protowire.SizeTag(2), "// size = protowire.SizeTag(1) + protowire.SizeTag(2)")
	//
	genFieldSizeFunc(g, field.Message.Fields[0], "mk", "size")
	genFieldSizeFunc(g, field.Message.Fields[1], "mv", "size")

}

func genFieldSizeFunc(g *protogen.GeneratedFile, field *protogen.Field, vname, size string) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P(size, "+= ", 1)
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		g.P(size, "+= protowire.SizeVarint(uint64(", vname, "))")
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		g.P(size, "+= protowire.SizeVarint(protowire.EncodeZigZag(int64(", vname, ")))")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		g.P(size, "+= 4")
	case protoreflect.FloatKind:
		g.P(size, "+= 4")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		g.P(size, "+= 8")
	case protoreflect.DoubleKind:
		g.P(size, "+= 8")
	case protoreflect.StringKind:
		g.P(size, "+= protowire.SizeBytes(len(", vname, "))")
	case protoreflect.BytesKind:
		g.P(size, "+= protowire.SizeBytes(len(", vname, "))")
	case protoreflect.MessageKind:
		g.P(size, "+= ", vname, ".MarshalSize()")
	case protoreflect.GroupKind:
		// unsupport
	}
	return //  goType, pointer
}
