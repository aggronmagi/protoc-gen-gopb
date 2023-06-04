package main

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func genMessageMarshal(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) {
	buf := make([]byte, 0, 32)
	g.P("// MarshalObjectTo marshal data to []byte")
	g.P("func (x *", m.GoIdent, ") MarshalObjectTo(buf []byte)(data []byte, err error) {")
	g.P("data = buf")
	for _, field := range m.Fields {
		genMessageMarshalField(g, field, buf)
	}
	g.P("return")
	g.P("}")
	g.P()

	g.P("// MarshalObject marshal data to []byte")
	g.P("func (x *", m.GoIdent, ") MarshalObject()(data []byte, err error) {")
	g.P("data = make([]byte, 0, x.MarshalSize())")
	g.P("return x.MarshalObjectTo(data)")
	g.P("}")
	g.P()

	if len(m.Fields) > 0 {
		g.Import(protogen.GoImportPath(cfg.wirepkg))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "VarintType", GoImportPath: protogen.GoImportPath(cfg.wirepkg)})
		g.Import(protogen.GoImportPath("errors"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "New", GoImportPath: "errors"})
	}
}

func genMessageMarshalField(g *protogen.GeneratedFile, field *protogen.Field, cache []byte) {
	// 数组
	if field.Desc.IsList() {
		genMarshalListField(g, field, cache)
		return
	}
	// map
	if field.Desc.IsMap() {
		genMarshalMapField(g, field, cache)
		return
	}
	// 序列化基础数据类型
	genMarshalBasicField(g, field, cache, "x."+field.GoName, true)
}

func genMarshalBasicField(g *protogen.GeneratedFile, field *protogen.Field, cache []byte, vname string, canIgnore bool) {
	//pointer = field.Desc.HasPresence()
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		if canIgnore {
			g.P("if ", vname, "{")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.VarintType)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.VarintType) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendVarint(data, protowire.EncodeBool(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.VarintType)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.VarintType) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendVarint(data, uint64(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.VarintType)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.VarintType) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendVarint(data, protowire.EncodeZigZag(int64(", vname, ")))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.Fixed32Type)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.Fixed32Type) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendFixed32(data, uint32(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.FloatKind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.Fixed32Type)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.Fixed32Type) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendFixed32(data, math.Float32bits(", vname, "))")
		if canIgnore {
			g.P("}")
		}
		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.Fixed64Type)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.Fixed64Type) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendFixed64(data, uint64(", vname, "))")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.DoubleKind:
		if canIgnore {
			g.P("if ", vname, " != 0 {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.Fixed64Type)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.Fixed64Type) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendFixed64(data, math.Float64bits(", vname, "))")
		if canIgnore {
			g.P("}")
		}
		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		if canIgnore {
			g.P("if len(", vname, ") > 0{")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.BytesType)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.BytesType) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendString(data, ", vname, ")")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.BytesKind:
		if canIgnore {
			g.P("if len(", vname, ") > 0{")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.BytesType)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.BytesType) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendBytes(data, ", vname, ")")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.MessageKind:
		//goType = "*" + g.QualifiedGoIdent(field.Message.GoIdent)
		//pointer = false // pointer captured as part of the type
		if canIgnore {
			g.P("if ", vname, " != nil {")
		}
		cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.BytesType)
		g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.BytesType) => ", BinaryBytes(cache))
		g.P("data = append(data, ", SourceBytes(cache), ")")
		g.P("data = protowire.AppendVarint(data, uint64(", vname, ".MarshalSize()))")
		g.P("data,err = ", vname, ".MarshalObjectTo(data)")
		g.P("if err != nil {")
		g.P("return")
		g.P("}")
		if canIgnore {
			g.P("}")
		}
	case protoreflect.GroupKind:
		// unsupport
	}
	return //  goType, pointer
}

func genMarshalListField(g *protogen.GeneratedFile, field *protogen.Field, cache []byte) {

	g.P("if len(x.", field.GoName, ") > 0{")
	defer g.P("}")

	// if field.Desc.Kind() == protoreflect.StringKind || field.Desc.Kind() == protoreflect.BytesKind || field.Desc.Kind() == protoreflect.MessageKind {
	// 	log.Println("repeated ", field.Desc.Kind(), ": packed:", field.Desc.IsPacked())
	// }

	// packed=false ,每个元素单独发送
	if !field.Desc.IsPacked() {
		g.P("for _,item := range x.", field.GoName, "{")
		genMarshalBasicField(g, field, cache, "item", false)
		g.P("}")
		return
	}

	// packed=true 元素一起发送
	cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.BytesType)
	g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.BytesType) => ", BinaryBytes(cache))
	g.P("data = append(data, ", SourceBytes(cache), ")")

	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P("data = protowire.AppendVarint(data, uint64(len(x.", field.GoName, ")))")
		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendVarint(data, protowire.EncodeBool(v))")
		g.P("}")
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		g.P("size := 0")
		g.P("for _,v := range x.", field.GoName, " {")
		g.P("size += protowire.SizeVarint(uint64(v))")
		g.P("}")
		g.P("data = protowire.AppendVarint(data, uint64(size))")

		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendVarint(data, uint64(v))")
		g.P("}")
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		g.P("size := 0")
		g.P("for _,v := range x.", field.GoName, " {")
		g.P("size += protowire.SizeVarint(protowire.EncodeZigZag(int64(v)))")
		g.P("}")
		g.P("data = protowire.AppendVarint(data, uint64(size))")

		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendVarint(data, protowire.EncodeZigZag(int64(v)))")
		g.P("}")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		g.P("data = protowire.AppendVarint(data, uint64(4*len(x.", field.GoName, ")))")
		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendFixed32(data, uint32(v))")
		g.P("}")
	case protoreflect.FloatKind:
		g.P("data = protowire.AppendVarint(data, uint64(4*len(x.", field.GoName, ")))")
		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendFixed32(data, math.Float32bits(v))")
		g.P("}")
		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float32bits", GoImportPath: "math"})
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		g.P("data = protowire.AppendVarint(data, uint64(8*len(x.", field.GoName, ")))")
		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendFixed64(data, uint64(v))")
		g.P("}")
	case protoreflect.DoubleKind:
		g.P("data = protowire.AppendVarint(data, uint64(8*len(x.", field.GoName, ")))")
		g.P("for _,v := range x.", field.GoName, "{")
		g.P("data = protowire.AppendFixed64(data,  math.Float64bits(v))")
		g.P("}")
		g.Import(protogen.GoImportPath("math"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Float64bits", GoImportPath: "math"})
	case protoreflect.StringKind:
		// 不存在packed=true
		// g.P("size := 0")
		// g.P("for _,v := range x.", field.GoName, " {")
		// g.P("size += protowire.SizeBytes(len(v))")
		// g.P("}")
		// g.P("data = protowire.AppendVarint(data, uint64(size))")

		// g.P("for _,v := range x.", field.GoName, "{")
		// g.P("data = protowire.AppendString(data, v)")
		// g.P("}")

		g.P("for _,item := range x.", field.GoName, "{")
		genMarshalBasicField(g, field, cache, "item", false)
		g.P("}")
	case protoreflect.BytesKind:
		// g.P("size := 0")
		// g.P("for _,v := range x.", field.GoName, " {")
		// g.P("size += protowire.SizeBytes(len(v))")
		// g.P("}")
		// g.P("data = protowire.AppendVarint(data, uint64(size))")

		// g.P("for _,v := range x.", field.GoName, "{")
		// g.P("data = protowire.AppendBytes(data, v)")
		// g.P("}")
		g.P("for _,item := range x.", field.GoName, "{")
		genMarshalBasicField(g, field, cache, "item", false)
		g.P("}")
	case protoreflect.MessageKind:
		// g.P("size := 0")
		// g.P("for _,v := range x.", field.GoName, " {")
		// g.P("size += v.MarshalSize()")
		// g.P(")}")
		// g.P("data = protowire.AppendVarint(data, uint64(size))")

		// g.P("for _,v := range x.", field.GoName, "{")
		// g.P("data,err = v.MarshalObjectTo(data)")
		// g.P("if err != nil {")
		// g.P("return")
		// g.P("}")
		// g.P("}")
		g.P("for _,item := range x.", field.GoName, "{")
		genMarshalBasicField(g, field, cache, "item", false)
		g.P("}")
	case protoreflect.GroupKind:
		// unsupport
	}
	return //  goType, pointer
}

func genMarshalMapField(g *protogen.GeneratedFile, field *protogen.Field, cache []byte) {
	g.P("if len(x.", field.GoName, ") > 0{")
	defer g.P("}")

	g.P("for mk,mv := range x.", field.GoName, " {")
	defer g.P("}")

	// tag 头
	cache = protowire.AppendTag(cache[:0], field.Desc.Number(), protowire.BytesType)
	g.P("// data = protowire.AppendTag(data, ", field.Desc.Number(), ", protowire.BytesType) => ", BinaryBytes(cache))

	g.P("data = append(data, ", SourceBytes(cache), ")")
	// 长度计
	g.P("size := ", protowire.SizeTag(1), "+", protowire.SizeTag(2), "// size = protowire.SizeTag(1) + protowire.SizeTag(2)")
	//
	genFieldSizeFunc(g, field.Message.Fields[0], "mk", "size", false)
	genFieldSizeFunc(g, field.Message.Fields[1], "mv", "size", false)

	g.P("data = protowire.AppendVarint(data, uint64(size))")
	genMarshalBasicField(g, field.Message.Fields[0], cache, "mk", false)
	genMarshalBasicField(g, field.Message.Fields[1], cache, "mv", false)
}

func SourceBytes(in []byte) string {
	buf := make([]byte, 0, len(in)*5-1)
	for k, v := range in {
		if k > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, '0', 'x')
		buf = strconv.AppendInt(buf, int64(v), 16)
	}
	return *(*string)(unsafe.Pointer(&buf))
}

func BinaryBytes(in []byte) string {
	// buf := make([]byte, 0, len(in)*9-1)
	// for k, v := range in {
	// 	if k > 0 {
	// 		buf = append(buf, ' ')
	// 	}
	// 	buf = strconv.AppendInt(buf, int64(v), 2)
	// }
	// return *(*string)(unsafe.Pointer(&buf))
	buf := strings.Builder{}
	buf.Grow(len(in) * 9)
	for k, v := range in {
		if k > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(fmt.Sprintf("%08b", v))
	}
	return buf.String()
}
