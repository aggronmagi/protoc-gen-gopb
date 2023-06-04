# protoc-gen-gopb
generate go code for protobuf

protoc-gen-gopb 从proto定义文件生成go结构体,并使用[protobuf-wire](https://protobuf.dev/programming-guides/encoding/)格式,进行序列化和反序列化. 

proto-gen-gopb 不支持 `oneof`,`weak`,`group`, 不支持默认值.

如果需要proto的反射,动态消息生成等, 请使用 `google.golang.org/protobuf/`

protoc-gen-gopb 使用 `google.golang.org/protobuf/encoding/protowire` 进行序列化和反序列化. 除此之外不再依赖任何pb相关的包. 可以使用 pwwire参数替换成本地的包. 

生成的代码,没有String方法. 没有实现proto.Message 接口. 也不能使用proto.Marshal/proto.Unmarshal . 


如果有需要,请自行定义 Message 及相关接口. 例:
``` go
// 生成的消息都会实现的接口.
type Message interface {
  MarshalObjectTo(buf []byte) (data []byte, err error) 
  MarshalObject() (data []byte, err error)
  UnmarshalObject(data []byte) (err error)
}

func Marshal(v Message) ([]byte, error) {
	return v.MarshalObject()
}

func Unmarshal(data []byte, v Message) (err error) {
	return v.UnmarshalObject(data)
}
```
## 参数
| 参数   | 环境变量          | 默认值                                          |
|--------|-------------------|-------------------------------------------------|
| pbwire | GOPB_WIRE_PACKAGE | "google.golang.org/protobuf/encoding/protowire" |
| get    | GOPB_GEN_GET      | false                                           |
| zap    | GOPB_GEN_ZAP      | true                                            |
|        | GOPB_GEN_DEBUG    | true                                           |

pbwire 用于替换引入序列化包的包名. 

get 是否生成Getter方法. 

zap 是否生成对应zap方法. 

GOPB_GEN_DEBUG: 生成的解析代码中,添加类型判定. 

## 生成代码预览
``` protobuf
message Example {
    int32 filed = 1;
}
```
``` go

type Example struct {
	Filed int32 `json:"filed,omitempty" db:"filed"`
}

func (x *Example) Reset() {
	*x = Example{}
}

// MarshalObjectTo marshal data to []byte
func (x *Example) MarshalObjectTo(buf []byte) (data []byte, err error) {
	data = buf
	if x.Filed != 0 {
		// data = protowire.AppendTag(data, 1, protowire.VarintType) => 00001000
		data = append(data, 0x8)
		data = protowire.AppendVarint(data, uint64(x.Filed))
	}
	return
}

// MarshalObject marshal data to []byte
func (x *Example) MarshalObject() (data []byte, err error) {
	data = make([]byte, 0, x.MarshalSize())
	return x.MarshalObjectTo(data)
}

// UnmarshalObject unmarshal data from []byte
func (x *Example) UnmarshalObject(data []byte) (err error) {
	index := 0
	ignoreGroup := 0
	for index < len(data) {
		num, typ, cnt := protowire.ConsumeTag(data[index:])
		if num == 0 {
			err = errors.New("invalid tag")
			return
		}

		index += cnt
		/// other code ...
		switch num {
		case 1:
			if typ != protowire.VarintType {
				err = errors.New("invlaid field Example.Filed id:1. not varint type")
				return
			}
			v, cnt := protowire.ConsumeVarint(data[index:])
			if cnt < 1 {
				err = errors.New("invlaid field Example.Filed id:1. invalid varint value")
				return
			}
			index += cnt
			x.Filed = int32(v)
		default: // skip fields
			/// other code ...
		}
	}

	return
}

// MarshalSize calc marshal data need space
func (x *Example) MarshalSize() (size int) {
	if x.Filed != 0 {
		size += 1 // size += protowire.SizeTag(,1)
		size += protowire.SizeVarint(uint64(x.Filed))
	}
	return
}

func (x *Example) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt32("Filed", x.Filed)
	return nil
}

```

## 压测结果:
``` shell
➜  benchpb git:(main) ✗ go test -bench=. -benchmem 
goos: darwin
goarch: amd64
pkg: benchpb
cpu: VirtualApple @ 2.50GHz
BenchmarkVS/complex-google-marshal-10              52545             22522 ns/op           11456 B/op        351 allocs/op
BenchmarkVS/complex-gopb-marshal-10               288487              3848 ns/op            1408 B/op          1 allocs/op
BenchmarkVS/complex-google-unmarshal-10            55656             21374 ns/op            8664 B/op        343 allocs/op
BenchmarkVS/complex-gopb-unmarshal-10             183645              6504 ns/op            6952 B/op        108 allocs/op
BenchmarkVS/basic_type-google-marshal-10         3146823               377.6 ns/op            96 B/op          1 allocs/op
BenchmarkVS/basic_type-gopb-marshal-10           9007825               130.8 ns/op            96 B/op          1 allocs/op
BenchmarkVS/basic_type-google-unmarshal-10       2410054               499.8 ns/op           960 B/op          4 allocs/op
BenchmarkVS/basic_type-gopb-unmarshal-10         3263012               373.0 ns/op           920 B/op          4 allocs/op
BenchmarkVS/repeated-google-marshal-10           1567020               761.8 ns/op           352 B/op          1 allocs/op
BenchmarkVS/repeated-gopb-marshal-10             3097495               383.3 ns/op           352 B/op          1 allocs/op
BenchmarkVS/repeated-google-unmarshal-10          584644              1984 ns/op            1848 B/op         49 allocs/op
BenchmarkVS/repeated-gopb-unmarshal-10            928567              1242 ns/op            1640 B/op         33 allocs/op
BenchmarkVS/map-int-google-marshal-10             136934              8714 ns/op            5328 B/op        137 allocs/op
BenchmarkVS/map-int-gopb-marshal-10               791127              1489 ns/op             352 B/op          1 allocs/op
BenchmarkVS/map-int-google-unmarshal-10           178422              6676 ns/op            4168 B/op        114 allocs/op
BenchmarkVS/map-int-gopb-unmarshal-10             512524              2289 ns/op            3656 B/op         38 allocs/op
BenchmarkVS/map-all-google-marshal-10              97893             12205 ns/op            5712 B/op        215 allocs/op
BenchmarkVS/map-all-gopb-marshal-10               609843              1978 ns/op             640 B/op          1 allocs/op
BenchmarkVS/map-all-google-unmarshal-10           111752             10664 ns/op            4384 B/op        179 allocs/op
BenchmarkVS/map-all-gopb-unmarshal-10             410235              2873 ns/op            3416 B/op         36 allocs/op
PASS
ok      benchpb 28.027s
```

压测代码: https://github.com/aggronmagi/benchpb

## protobuf-wire 记录

参照 https://protobuf.dev/programming-guides/encoding/

字段类型:
| ID | Name   | Used For                                                 |                                         |
|----+--------+----------------------------------------------------------+-----------------------------------------|
|  0 | VARINT | int32, int64, uint32, uint64, sint32, sint64, bool, enum | varint 编码                             |
|  1 | I64    | fixed64, sfixed64, double                                | 定长8个字节                             |
|  2 | LEN    | string, bytes, embedded messages, packed repeated fields | 先跟一个varint编码的长度. 后面是payload |
|  3 | SGROUP | group start (deprecated)                                 |                                         |
|  4 | EGROUP | group end (deprecated)                                   |                                         |
|  5 | I32    | fixed32, sfixed32, float                                 | 定长4个字节                             |

bool,enum 都使用 varint. 

sint32,sint64 都使用了zigzag. 

sfix32,sfix64 未使用zigzag.

repeated 字段. proto3 默认开启packed=true. proto2需要手动开启. 

repeated 字段. packed=true时候,外层使用len,内部连续数据元素. 但是 对 string,bytes, packed=true 不生效. 因为他们正常就是使用len类型. 

map 相当与repeated message. 如下;

``` protobuf
message Test6 {
  map<string, int32> g = 7;
}
```
``` protobuf
message Test6 {
  message g_Entry {
    optional string key = 1;
    optional int32 value = 2;
  }
  repeated g_Entry g = 7;
}
```
g_Entry 解析需要支持乱序的. 比如先发的字段2,后发的字段1. 


