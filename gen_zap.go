package main

import (
	"fmt"
	"log"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func genZapMessage(g *protogen.GeneratedFile, m *protogen.Message) {
	if m.Desc.IsMapEntry() {
		return
	}
	// object marshal
	//g.Annotate(m.GoIdent.GoName, m.Location)
	g.Import(protogen.GoImportPath("go.uber.org/zap/zapcore"))
	g.QualifiedGoIdent(protogen.GoIdent{GoName: "Abc", GoImportPath: "go.uber.org/zap/zapcore"})
	g.P("func (x *", m.GoIdent, ") MarshalLogObject(enc zapcore.ObjectEncoder) error {")
	for _, field := range m.Fields {
		if field.Desc.IsWeak() {
			continue
		}

		keyName := field.GoName
		fieldName := field.GoName
		if field.Desc.IsMap() {
			g.P(`enc.AddObject("`, keyName, `", zapcore.ObjectMarshalerFunc(func(oe zapcore.ObjectEncoder) error {`)
			g.P(`for k,v := range x.`, fieldName, "{")
			funcName, fieldMethod := getZapFieldFunc(field.Message.Fields[1])
			g.P(fmt.Sprintf(`enc.Add%s(%s, v%s)`, funcName, getZaoFieldMapKey(g, field.Message.Fields[0]), fieldMethod))
			g.P("}")
			g.P("return nil")
			g.P("}))")
			continue
		}
		funcName, fieldMethod := getZapFieldFunc(field)
		switch {
		case field.Desc.IsList():
			g.P(fmt.Sprintf(`enc.AddArray("%s", zapcore.ArrayMarshalerFunc(func(ae zapcore.ArrayEncoder) error {`, keyName))
			g.P("for _,v := range x.", fieldName, "{")
			if funcName == "Binary" {
				g.Import(protogen.GoImportPath("encoding/base64"))
				g.QualifiedGoIdent(protogen.GoIdent{GoName: "NewEncodeToString", GoImportPath: "encoding/base64"})

				g.P(fmt.Sprintf("ae.AppendString(base64.StdEncoding.EncodeToString(v%s))", fieldMethod))
			} else {
				g.P(fmt.Sprintf("ae.Append%s(v%s)", funcName, fieldMethod))
			}
			g.P("}")
			g.P("return nil")
			g.P("}))")
		default:
			g.P(fmt.Sprintf(`enc.Add%s("%s", x.%s%s)`, funcName, keyName, fieldName, fieldMethod))
		}
	}
	g.P("return nil")
	g.P("}")
	g.P()

	g.P("type ZapArray", m.GoIdent, " []*", m.GoIdent)
	g.P("func (x ZapArray", m.GoIdent, ")  MarshalLogArray(ae zapcore.ArrayEncoder) error {")
	g.P("for _, v := range x {")
	g.P("ae.AppendObject(v)")
	g.P("}")
	g.P("return nil")
	g.P("}")
	g.Import(protogen.GoImportPath("go.uber.org/zap"))
	g.QualifiedGoIdent(protogen.GoIdent{GoName: "Abc", GoImportPath: "go.uber.org/zap"})
	g.P(`func LogArray`, m.GoIdent, `(name string, v []*`, m.GoIdent, `) zap.Field {`)
	g.P(`return zap.Array(name, ZapArray`, m.GoIdent, `(v))`)
	g.P("}")

	// array marshal
}

func getZaoFieldMapKey(g *protogen.GeneratedFile, field *protogen.Field) (funcName string) {
	isImport := true
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		funcName = "strconv.FormatBool(k)"
	case protoreflect.EnumKind:
		funcName = "k.String()"
		isImport = false
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		funcName = "strconv.FormatInt(int64(k), 10)"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		funcName = "strconv.FormatUint(uint64(k), 10)"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		funcName = "strconv.FormatInt(k, 10)"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		funcName = "strconv.FormatUint(k, 10)"
	case protoreflect.FloatKind:
		funcName = "strconv.FormatFloat32(k, 10)"
	case protoreflect.DoubleKind:
		funcName = "strconv.FormatFloat64(k, 10)"
	case protoreflect.StringKind:
		funcName = "k"
		isImport = false
	case protoreflect.BytesKind:
		funcName = "{Invalid Map Key - []byte}"
		log.Printf("invalid map key type []byte. %#v", field)
	case protoreflect.MessageKind, protoreflect.GroupKind:
		funcName = "{Invalid Map Key - Object}"
		log.Printf("invalid map key type object. %#v", field)
	}
	if isImport {
		g.Import(protogen.GoImportPath("strconv"))
		g.QualifiedGoIdent(protogen.GoIdent{GoName: "Abc", GoImportPath: "strconv"})
	}
	return
}

func getZapFieldFunc(field *protogen.Field) (funcName, fieldMethod string) {
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		funcName = "Bool"
	case protoreflect.EnumKind:
		funcName = "String"
		fieldMethod = ".String()"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		funcName = "Int32"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		funcName = "Uint32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		funcName = "Int64"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		funcName = "Uint64"
	case protoreflect.FloatKind:
		funcName = "Float32"
	case protoreflect.DoubleKind:
		funcName = "Float64"
	case protoreflect.StringKind:
		funcName = "String"
	case protoreflect.BytesKind:
		funcName = "Binary"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		funcName = "Object"
	}
	return
}
