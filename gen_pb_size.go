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
	genSizeBasicField(g, field, "x."+field.GoName, true)
}

func genSizeBasicField(g *protogen.GeneratedFile, field *protogen.Field, vname string, canIgnore bool) {
	//pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		if canIgnore {
			g.P("if ", vname, "{")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.FloatKind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.DoubleKind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		if canIgnore {
			g.P("if len(", vname, ") > 0{")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.BytesKind:
		if canIgnore {
			g.P("if len(", vname, ") > 0{")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.MessageKind:
		//goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		//pointer = false // pointer captured as part of the type

		if canIgnore {
			g.P("if ", vname, " != nil {")
		}
		g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
		genFieldSizeFunc(g, field, vname, "size", false)
		if canIgnore {
			g.P("}")
		}
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
		genSizeBasicField(g, field, "item", false)
		g.P("}")
		return
	}

	// packed=true 元素一起发送
	g.P("size += ", protowire.SizeTag(field.Desc.Number()), "// size += protowire.SizeTag(,", field.Desc.Number(), ")")
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("size += protowire.SizeBytes(len(x.", field.GoName, "))")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind:
		g.P("size += protowire.SizeBytes(len(x.", field.GoName, ")*4)")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		g.P("size += protowire.SizeBytes(len(x.", field.GoName, ")*8)")
	case protoreflect.BytesKind, protoreflect.StringKind, protoreflect.MessageKind:
		// NOTE: 不支持packed
		g.P("for _,item := range x.", field.GoName, "{")
		genSizeBasicField(g, field, "item", false)
		g.P("}")
	default:
		genSliceFieldSizeFunc(g, field, "x."+field.GoName, "size")
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
	g.P("msize := ", protowire.SizeTag(1), "+", protowire.SizeTag(2), "// size = protowire.SizeTag(1) + protowire.SizeTag(2)")
	//
	genFieldSizeFunc(g, field.Message.Fields[0], "mk", "msize", false)
	genFieldSizeFunc(g, field.Message.Fields[1], "mv", "msize", false)
	//
	g.P("size += protowire.SizeBytes(msize)")

}

func genFieldSizeFunc(g *protogen.GeneratedFile, field *protogen.Field, vname, size string, canIgnore bool) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		if canIgnore {
			g.P("if ", vname, "{")
		}
		g.P(size, "+= ", 1)
		if canIgnore {
			g.P("}")
		}
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P(size, "+= protowire.SizeVarint(uint64(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P(size, "+= protowire.SizeVarint(protowire.EncodeZigZag(int64(", vname, ")))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P(size, "+= 4")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.FloatKind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P(size, "+= 4")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P(size, "+= 8")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.DoubleKind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		g.P(size, "+= 8")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.StringKind:
		if canIgnore {
			g.P("if len(", vname, ") > 0 {")
		}
		g.P(size, "+= protowire.SizeBytes(len(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.BytesKind:
		if canIgnore {
			g.P("if len(", vname, ") > 0 {")
		}
		g.P(size, "+= protowire.SizeBytes(len(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.MessageKind:
		if canIgnore {
			g.P("if ", vname, " != nil {")
		}
		g.P(size, "+= protowire.SizeBytes(", vname, ".MarshalSize())")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.GroupKind:
		// unsupport
	}
	return //  goType, pointer
}

func genSliceFieldSizeFunc(g *protogen.GeneratedFile, field *protogen.Field, vname, size string) {
	switch field.Desc.Kind() {
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		g.P("if len(", vname, ") > 0 { fsize :=0 ")
		g.P("for _,item := range ", vname, "{")
		g.P("fsize += protowire.SizeVarint(uint64(item))")
		g.P("}")
		g.P(size, "+= protowire.SizeBytes(fsize)")
		g.P("}")
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		g.P("if len(", vname, ") > 0 { fsize :=0 ")
		g.P("for _,item := range ", vname, "{")
		g.P("fsize += protowire.SizeVarint(protowire.EncodeZigZag(int64(item)))")
		g.P("}")
		g.P(size, "+= protowire.SizeBytes(fsize)")
		g.P("}")
	case protoreflect.StringKind, protoreflect.BytesKind:
		g.P("if len(", vname, ") > 0 { fsize :=0 ")
		g.P("for _,item := range ", vname, "{")
		g.P("fsize += protowire.SizeBytes(len(item))")
		g.P("}")
		g.P(size, "+= protowire.SizeBytes(fsize)")
		g.P("}")
	case protoreflect.MessageKind:
		g.P("if len(", vname, ") > 0 { fsize :=0 ")
		g.P("for _,item := range ", vname, "{")
		g.P("fsize += protowire.SizeBytes(item.MarshalSize())")
		g.P("}")
		g.P(size, "+= protowire.SizeBytes(fsize)")
		g.P("}")
	case protoreflect.GroupKind:
		// unsupport
	}
	return //  goType, pointer
}
