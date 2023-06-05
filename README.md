# protoc-gen-gopb
generate go code for protobuf

protoc-gen-gopb 从proto定义文件生成go结构体,并使用[protobuf-wire](https://protobuf.dev/programming-guides/encoding/)格式,进行序列化和反序列化. 

proto-gen-gopb 不支持 `oneof`,`weak`,`group`, 不支持默认值. 

如果需要proto的反射,动态消息生成等, 请使用 `google.golang.org/protobuf/`.

如果你追求完整的protobuf功能,可以使用gogo/protobuf, 其中 gogofaster比gopb更适合你. 

如果你只是想使用,不想定制, gogo/protobuf 可能更适合你. 

gopb 主要是让你了解pb序列化,反序列化的细节,在此基础上可以你可以自己实现,扩展更多功能,而且能保证和主流的protobuf-wire协议兼容. 

protoc-gen-gopb 使用 `google.golang.org/protobuf/encoding/protowire` 进行序列化和反序列化. 除此之外不再依赖任何pb相关的包. 可以使用 pbwire参数替换成本地的包. 

生成的代码,没有String方法. 没有实现proto.Message 接口. 也不能使用proto.Marshal/proto.Unmarshal . 

如果有需要,请自行定义 Message 及相关接口. 例:
``` go
// 生成的消息都会实现的接口.
type Message interface {
  MarshalObjectTo(buf []byte) (data []byte, err error) 
  MarshalObject() (data []byte, err error)
  UnmarshalObject(data []byte) (err error)
  MarshalSize() (size int)
}

func Marshal(v Message) ([]byte, error) {
	return v.MarshalObject()
}

func Unmarshal(data []byte, v Message) (err error) {
	return v.UnmarshalObject(data)
}
```
## 可能的定制
只是举个例子, 具体怎么定制, 需要你根据实际需求去决定.
 - 比如 某个字段是 slice/map ,在生成解析时候, 定制长度,以减少内存分配次数.(因为你设计pb时候,可能是知道所需长度的.)
 - 跨节点,增量数据更新. 在接收数据的一方,提供hook接口,监听变动. 

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
			if typ != protowire.VarintType { // 这个条件判断就是 GOPB_GEN_DEBUG 加的.
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

压测代码: https://github.com/aggronmagi/benchpb 

与 google.golang.org/protobuf/ 相比
``` shell
➜  benchpb git:(main) ✗ go test -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/aggronmagi/benchpb
cpu: VirtualApple @ 2.50GHz
BenchmarkVS/complex-google-marshal-10         	   53272	     21991 ns/op	   11456 B/op	     351 allocs/op
BenchmarkVS/complex-gopb-marshal-10           	  290721	      3825 ns/op	    1408 B/op	       1 allocs/op
BenchmarkVS/complex-google-unmarshal-10       	   58266	     20423 ns/op	    8672 B/op	     343 allocs/op
BenchmarkVS/complex-gopb-unmarshal-10         	  187006	      6365 ns/op	    6952 B/op	     108 allocs/op
BenchmarkVS/basic_type-google-marshal-10      	 3151597	       380.3 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-gopb-marshal-10        	 9059457	       130.3 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-google-unmarshal-10    	 2441830	       492.9 ns/op	     960 B/op	       4 allocs/op
BenchmarkVS/basic_type-gopb-unmarshal-10      	 3327951	       364.1 ns/op	     920 B/op	       4 allocs/op
BenchmarkVS/repeated-google-marshal-10        	 1555779	       810.2 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/repeated-gopb-marshal-10          	 3056652	       384.2 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/repeated-google-unmarshal-10      	  591796	      1954 ns/op	    1848 B/op	      49 allocs/op
BenchmarkVS/repeated-gopb-unmarshal-10        	  959810	      1219 ns/op	    1640 B/op	      33 allocs/op
BenchmarkVS/map-int-google-marshal-10         	  136380	      8776 ns/op	    5328 B/op	     137 allocs/op
BenchmarkVS/map-int-gopb-marshal-10           	  774550	      1506 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/map-int-google-unmarshal-10       	  177134	      6842 ns/op	    4168 B/op	     114 allocs/op
BenchmarkVS/map-int-gopb-unmarshal-10         	  505020	      2292 ns/op	    3656 B/op	      38 allocs/op
BenchmarkVS/map-all-google-marshal-10         	   96702	     12118 ns/op	    5712 B/op	     215 allocs/op
BenchmarkVS/map-all-gopb-marshal-10           	  595708	      1971 ns/op	     640 B/op	       1 allocs/op
BenchmarkVS/map-all-google-unmarshal-10       	  109795	     10941 ns/op	    4376 B/op	     179 allocs/op
BenchmarkVS/map-all-gopb-unmarshal-10         	  410346	      2827 ns/op	    3416 B/op	      36 allocs/op
PASS
ok  	github.com/aggronmagi/benchpb	28.022s
```

与gogo/prorobuf相比

``` shell
goos: darwin
goarch: amd64
pkg: github.com/aggronmagi/benchpb/gogofaster
cpu: VirtualApple @ 2.50GHz
BenchmarkVS/complex-gogofaster-marshal-10         	  237554	      4324 ns/op	    1736 B/op	       8 allocs/op
BenchmarkVS/complex-gopb-marshal-10               	  307136	      3768 ns/op	    1408 B/op	       1 allocs/op
BenchmarkVS/complex-gogofaster-unmarshal-10       	  214299	      5616 ns/op	    6760 B/op	     103 allocs/op
BenchmarkVS/complex-gopb-unmarshal-10             	  192345	      6174 ns/op	    6952 B/op	     108 allocs/op
BenchmarkVS/basic_type-gogofaster-marshal-10      	 7614111	       156.0 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-gopb-marshal-10            	 9148363	       129.6 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-gogofaster-unmarshal-10    	 3830631	       314.5 ns/op	     920 B/op	       4 allocs/op
BenchmarkVS/basic_type-gopb-unmarshal-10          	 3586162	       342.4 ns/op	     920 B/op	       4 allocs/op
BenchmarkVS/repeated-gogofaster-marshal-10        	 2141361	       549.4 ns/op	     680 B/op	       8 allocs/op
BenchmarkVS/repeated-gopb-marshal-10              	 3180798	       372.8 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/repeated-gogofaster-unmarshal-10      	 1000000	      1126 ns/op	    1464 B/op	      28 allocs/op
BenchmarkVS/repeated-gopb-unmarshal-10            	  983776	      1182 ns/op	    1640 B/op	      33 allocs/op
BenchmarkVS/map-int-gogofaster-marshal-10         	  734912	      1613 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/map-int-gopb-marshal-10               	  794096	      1490 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/map-int-gogofaster-unmarshal-10       	  587643	      2014 ns/op	    3656 B/op	      38 allocs/op
BenchmarkVS/map-int-gopb-unmarshal-10             	  524802	      2182 ns/op	    3656 B/op	      38 allocs/op
BenchmarkVS/map-all-gogofaster-marshal-10         	  536385	      2204 ns/op	     640 B/op	       1 allocs/op
BenchmarkVS/map-all-gopb-marshal-10               	  581611	      1966 ns/op	     640 B/op	       1 allocs/op
BenchmarkVS/map-all-gogofaster-unmarshal-10       	  479342	      2481 ns/op	    3416 B/op	      36 allocs/op
BenchmarkVS/map-all-gopb-unmarshal-10             	  419162	      2777 ns/op	    3416 B/op	      36 allocs/op
PASS
ok  	github.com/aggronmagi/benchpb/gogofaster	27.331s
```

## protobuf-wire 记录

参照 https://protobuf.dev/programming-guides/encoding/

字段类型:
| ID | Name   | Used For                                                 |                                         |
|----|--------|----------------------------------------------------------|-----------------------------------------|
| 0  | VARINT | int32, int64, uint32, uint64, sint32, sint64, bool, enum | varint 编码                             |
| 1  | I64    | fixed64, sfixed64, double                                | 定长8个字节                             |
| 2  | LEN    | string, bytes, embedded messages, packed repeated fields | 先跟一个varint编码的长度. 后面是payload |
| 3  | SGROUP | group start (deprecated)                                 |                                         |
| 4  | EGROUP | group end (deprecated)                                   |                                         |
| 5  | I32    | fixed32, sfixed32, float                                 | 定长4个字节                             |

bool,enum 都使用 varint. 

sint32,sint64 都使用了zigzag. 

sfix32,sfix64 未使用zigzag.

repeated 字段. proto3 默认开启packed=true. proto2需要手动开启. 

repeated 字段. packed=true时候,外层使用len,内部连续数据元素. 但是 对 string,bytes以及message类型,选项 packed=true 不生效. 因为他们正常就是使用len类型. 

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
`message g_Entry` 解析需要支持乱序的. 比如先发的字段2,后发的字段1. 


