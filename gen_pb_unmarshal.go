package main

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func genMessageUnmarshal(g *protogen.GeneratedFile, f *protogen.File, m *protogen.Message) {
	g.P("// UnmarshalObject unmarshal data from []byte")
	g.P("func (x *", m.GoIdent, ") UnmarshalObject(data []byte)(err error) {")
	defer g.P("return\n}")
	if len(m.Fields) < 1 {
		return
	}
	g.P(`index := 0
	ignoreGroup := 0
	for index < len(data) {
		num, typ, cnt := protowire.ConsumeTag(data[index:])
		if num == 0 {
			err = errors.New("invalid tag")
			return
		}

		index += cnt
		// ignore group
		if ignoreGroup > 0 {
			switch typ {
			case protowire.VarintType:
				_, cnt := protowire.ConsumeVarint(data[index:])
				if cnt < 1 {
					err = protowire.ParseError(cnt)
					return
				}
				index += cnt
			case protowire.Fixed32Type:
				index += 4
			case protowire.Fixed64Type:
				index += 8
			case protowire.BytesType:
				v, cnt := protowire.ConsumeBytes(data[index:])
				if v == nil {
					if cnt < 0 {
						err = protowire.ParseError(cnt)
					} else {
						err = errors.New("invalid data")
					}
					return
				}
				index += cnt
			case protowire.StartGroupType:
				ignoreGroup++
			case protowire.EndGroupType:
				ignoreGroup--
			}
			continue
		}
		switch num {`)
	for _, field := range m.Fields {
		g.P("case ", field.Desc.Number(), ":")
		genUnmarshalField(g, m, field)
	}
	g.P("default: // skip fields")
	g.P(`	switch typ {
			case protowire.VarintType:
				_, cnt := protowire.ConsumeVarint(data[index:])
				if cnt < 1 {
					err = protowire.ParseError(cnt)
					return
				}
				index += cnt
			case protowire.Fixed32Type:
				index += 4
			case protowire.Fixed64Type:
				index += 8
			case protowire.BytesType:
				v, cnt := protowire.ConsumeBytes(data[index:])
				if v == nil {
					if cnt < 0 {
						err = protowire.ParseError(cnt)
					} else {
						err = errors.New("invalid data")
					}
					return
				}
				index += cnt
			case protowire.StartGroupType:
				ignoreGroup++
			case protowire.EndGroupType:
				ignoreGroup--
			}}`)

	g.P("}")
	g.P()
}

func genUnmarshalField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field) {
	if field.Desc.IsMap() {
		genUnmarshalMapField(g, m, field)
		return
	}
	if field.Desc.IsList() {
		genUnmarshalListField(g, m, field)
		return
	}
	genUnmarshalBasicField(g, m, field, "x."+field.GoName)
}
func genUnmarshalMapField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field) {
	goTyp, _ := fieldGoType(g, field)
	// map 必须支持乱序的k,v. 如果发送方是发送了packed的方式, 那么只使用最后的值.
	g.P(`if typ != protowire.BytesType {
		err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
		return
	}
	buf, cnt := protowire.ConsumeBytes(data[index:])
	if buf == nil {
		err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
		return
	}
	index += cnt
	if x.`, field.GoName, ` == nil {
		x.`, field.GoName, ` = make(`, goTyp, `)
	}`)
	keyTyp, _ := fieldGoType(g, field.Message.Fields[0])
	valTyp, _ := fieldGoType(g, field.Message.Fields[1])
	g.P("var mk ", keyTyp)
	g.P("var mv ", valTyp)
	//
	g.P("for sindex:=0; sindex<len(buf);{")
	g.P(`mi, typ, scnt := protowire.ConsumeTag(buf[sindex:])
		if scnt < 1 {
			err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
			return
		}
		sindex += scnt
		switch mi {`)
	g.P("case 1:")
	genUnmarshalMapBasicField(g, m, field.Message.Fields[0], "mk")
	g.P("case 2:")
	genUnmarshalMapBasicField(g, m, field.Message.Fields[1], "mv")
	g.P(`}`)
	g.P("}")
	g.P("x.", field.GoName, "[mk]=mv")
}
func genUnmarshalListField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field) {
	goTyp, _ := fieldGoType(g, field)
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		g.P(`// packed=false
			if typ == protowire.VarintType {
				v, cnt := protowire.ConsumeVarint(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `, protowire.DecodeBool(v))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make([]bool, 0, cnt)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeVarint(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `, protowire.DecodeBool(v))
			}`)
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		g.P(`// packed=false
			if typ == protowire.VarintType {
				v, cnt := protowire.ConsumeVarint(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(v))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, 2)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeVarint(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(v))
			}`)
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		g.P(`// packed=false
			if typ == protowire.VarintType {
				v, cnt := protowire.ConsumeVarint(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(protowire.DecodeZigZag(v)))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, 2)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeVarint(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(protowire.DecodeZigZag(v)))
			}`)
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		g.P(`// packed=false
			if typ == protowire.Fixed32Type {
				v, cnt := protowire.ConsumeFixed32(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(v))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, cnt/4)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeFixed32(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(v))
			}`)
	case protoreflect.FloatKind:
		g.P(`// packed=false
			if typ == protowire.Fixed32Type {
				v, cnt := protowire.ConsumeFixed32(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  math.Float32frombits(v))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, cnt/4)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeFixed32(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  math.Float32frombits(v))
			}`)
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		g.P(`// packed=false
			if typ == protowire.Fixed64Type {
				v, cnt := protowire.ConsumeFixed64(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(v))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, cnt/4)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeFixed64(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `, `, strings.TrimPrefix(goTyp, "[]"), `(v))
			}`)
	case protoreflect.DoubleKind:
		g.P(`// packed=false
			if typ == protowire.Fixed64Type {
				v, cnt := protowire.ConsumeFixed64(data[index:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
					return
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  math.Float64frombits(v))
				index += cnt
				continue
			}
			// packed = true 
			if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, cnt/8)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeFixed64(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  math.Float64frombits(v))
			}`)
	case protoreflect.StringKind:
		g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, 2)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeString(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  v)
			}`)
	case protoreflect.BytesKind:
		g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, 2)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeBytes(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  v)
			}`)
	case protoreflect.MessageKind:
		g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid repeated tag value")
				return
			}
			buf, cnt := protowire.ConsumeBytes(data[index:])
			if buf == nil {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt
			if x.`, field.GoName, ` == nil {
				x.`, field.GoName, ` = make(`, goTyp, `, 0, 2)
			}
			sub := 0
			for sub < len(buf) {
				v, cnt := protowire.ConsumeBytes(buf[sub:])
				if cnt < 1 {
					err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid item value")
					return
				}
				sub += cnt
				item := &`, strings.TrimPrefix(goTyp, "[]*"), `{}
				err = item.UnmarshalObject(v)
				if err != nil {
					return 
				}
				x.`, field.GoName, ` = append(x.`, field.GoName, `,  item)
			}`)
	case protoreflect.GroupKind:
		// unsupport
	}
}

func genUnmarshalBasicField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field, vname string) {
	goTyp, _ := fieldGoType(g, field)
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		if cfg.debug {
			g.P(`if typ != protowire.VarintType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not varint type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeVarint(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
				return
			}
			index += cnt`)
		g.P(vname, "= protowire.DecodeBool(v)")
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		if cfg.debug {
			g.P(`if typ != protowire.VarintType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not varint type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeVarint(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
				return
			}
			index += cnt`)
		g.P(vname, "= ", goTyp, "(v)")
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		if cfg.debug {
			g.P(`if typ != protowire.VarintType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not varint zigzag type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeVarint(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint zigzag value")
				return
			}
			index += cnt`)
		g.P(vname, "= ", goTyp, "(protowire.DecodeZigZag(v))")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed32Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i32 type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeFixed32(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i32 value")
				return
			}
			index += cnt`)
		g.P(vname, "= ", goTyp, "(v)")
	case protoreflect.FloatKind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed32Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i32 type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeFixed32(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i32 value")
				return
			}
			index += cnt`)
		g.P(vname, "= math.Float32frombits(v)")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed64Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i64 type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeFixed64(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i64 value")
				return
			}
			index += cnt`)
		g.P(vname, "= ", goTyp, "(v)")
	case protoreflect.DoubleKind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed64Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i64 type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeFixed64(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i64 value")
				return
			}
			index += cnt`)
		g.P(vname, "= math.Float64frombits(v)")
	case protoreflect.StringKind:
		if cfg.debug {
			g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not len type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeString(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt`)
		g.P(vname, "=v")
	case protoreflect.BytesKind:
		if cfg.debug {
			g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not len type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeBytes(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			index += cnt`)
		g.P(vname, "= make([]byte, len(v))")
		g.P("copy(", vname, ", v)")
	case protoreflect.MessageKind:
		if cfg.debug {
			g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not message type")
				return
			}`)
		}

		g.P(`v, cnt := protowire.ConsumeBytes(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid message value")
				return
			}
			index += cnt`)
		g.P(vname, " = &", strings.TrimPrefix(goTyp, "*"), "{}")
		g.P("err = ", vname, ".UnmarshalObject(v)")
		g.P(`if err != nil {return}`)
	case protoreflect.GroupKind:
		// unsupport
	}
	return
}

func genUnmarshalMapBasicField(g *protogen.GeneratedFile, m *protogen.Message, field *protogen.Field, vname string) {
	goTyp, _ := fieldGoType(g, field)
	switch field.Desc.Kind() {
	case protoreflect.BoolKind:
		if cfg.debug {
			g.P(`if typ != protowire.VarintType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not varint type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeVarint(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= protowire.DecodeBool(v)")
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		if cfg.debug {
			g.P(`if typ != protowire.VarintType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not varint type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeVarint(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= ", goTyp, "(v)")
	case protoreflect.Sint32Kind, protoreflect.Sint64Kind:
		if cfg.debug {
			g.P(`if typ != protowire.VarintType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not varint zigzag type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeVarint(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid varint zigzag value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= ", goTyp, "(protowire.DecodeZigZag(v))")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed32Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i32 type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeFixed32(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i32 value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= ", goTyp, "(v)")
	case protoreflect.FloatKind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed32Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i32 type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeFixed32(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i32 value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= math.Float32frombits(v)")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed64Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i64 type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeFixed64(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i64 value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= ", goTyp, "(v)")
	case protoreflect.DoubleKind:
		if cfg.debug {
			g.P(`if typ != protowire.Fixed64Type {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not i64 type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeFixed64(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid i64 value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= math.Float32frombits(v)")
	case protoreflect.StringKind:
		if cfg.debug {
			g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not len type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeString(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			sindex += scnt`)
		g.P(vname, "=v")
	case protoreflect.BytesKind:
		if cfg.debug {
			g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not len type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeBytes(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid len value")
				return
			}
			sindex += scnt`)
		g.P(vname, "= make([]byte, len(v))")
		g.P("copy(", vname, ", v)")
	case protoreflect.MessageKind:
		if cfg.debug {
			g.P(`if typ != protowire.BytesType {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. not message type")
				return
			}`)
		}

		g.P(`v, scnt := protowire.ConsumeBytes(buf[sindex:])
			if scnt < 1 {
				err = errors.New("invlaid field `, m.GoIdent, ".", field.GoName, ` id:`, field.Desc.Number(), `. invalid message value")
				return
			}
			sindex += scnt`)
		g.P(vname, " = &", strings.TrimPrefix(goTyp, "*"), "{}")
		g.P("err = ", vname, ".UnmarshalObject(v)")
		g.P(`if err != nil {return}`)
	case protoreflect.GroupKind:
		// unsupport
	}
	return
}
