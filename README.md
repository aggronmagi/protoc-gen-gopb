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
	Filed int32 `json:"filed,omitempty"`
}

func (x *Example) Reset() {
	*x = Example{}
}

// MarshalObject marshal data to []byte
func (x *Example) MarshalObject() (data []byte, err error) {
	data = make([]byte, 0, x.MarshalSize())
	return x.MarshalObjectTo(data)
}

// MarshalSize calc marshal data need space
func (x *Example) MarshalSize() (size int) {
	if x.Filed != 0 {
		// 1 = protowire.SizeTag(1)
		size += 1 + protowire.SizeVarint(uint64(x.Filed))
	}
	return
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

// UnmarshalObject unmarshal data from []byte
func (x *Example) UnmarshalObject(data []byte) (err error) {
	index := 0
	for index < len(data) {
		num, typ, cnt := protowire.ConsumeTag(data[index:])
		if num == 0 {
			err = errors.New("invalid tag")
			return
		}

		index += cnt
		switch num {
		case 1:
			v, cnt := protowire.ConsumeVarint(data[index:])
			if cnt < 1 {
				err = errors.New("parse Example.Filed ID:1 : invalid varint value")
				return
			}
			index += cnt
			x.Filed = int32(v)
		default: // skip fields
			cnt = protowire.ConsumeFieldValue(num, typ, data[index:])
			if cnt < 0 {
				return protowire.ParseError(cnt)
			}
			index += cnt
		}
	}

	return
}

func (x *Example) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt32("Filed", x.Filed)
	return nil
}

type ZapArrayExample []*Example

func (x ZapArrayExample) MarshalLogArray(ae zapcore.ArrayEncoder) error {
	for _, v := range x {
		ae.AppendObject(v)
	}
	return nil
}

func LogArrayExample(name string, v []*Example) zap.Field {
	return zap.Array(name, ZapArrayExample(v))
}

```

## 压测结果:

压测代码: https://github.com/aggronmagi/benchpb 

与 google.golang.org/protobuf/ 相比
``` shell
➜  benchpb git:(main) go test -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/aggronmagi/benchpb
cpu: VirtualApple @ 2.50GHz
BenchmarkVS/complex-google-marshal-10         	   51793	     22461 ns/op	   12096 B/op	     351 allocs/op
BenchmarkVS/complex-gopb-marshal-10           	  269382	      4151 ns/op	    2048 B/op	       1 allocs/op
BenchmarkVS/complex-google-unmarshal-10       	   52182	     22996 ns/op	   10136 B/op	     391 allocs/op
BenchmarkVS/complex-gopb-unmarshal-10         	  131690	      7820 ns/op	    8152 B/op	     151 allocs/op
BenchmarkVS/basic_type-google-marshal-10      	 2844948	       416.2 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-gopb-marshal-10        	 8264270	       142.3 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-google-unmarshal-10    	 1866696	       636.9 ns/op	    1472 B/op	       4 allocs/op
BenchmarkVS/basic_type-gopb-unmarshal-10      	 2653176	       455.9 ns/op	    1304 B/op	       4 allocs/op
BenchmarkVS/repeated-google-marshal-10        	 1479296	       807.4 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/repeated-gopb-marshal-10          	 3024024	       394.0 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/repeated-google-unmarshal-10      	  558592	      2106 ns/op	    2360 B/op	      49 allocs/op
BenchmarkVS/repeated-gopb-unmarshal-10        	  870282	      1354 ns/op	    2024 B/op	      33 allocs/op
BenchmarkVS/nopack-google-marshal-10          	 1401404	       835.7 ns/op	     416 B/op	       1 allocs/op
BenchmarkVS/nopack-gopb-marshal-10            	 3143082	       376.9 ns/op	     416 B/op	       1 allocs/op
BenchmarkVS/nopack-google-unmarshal-10        	  448196	      2664 ns/op	    2360 B/op	      49 allocs/op
BenchmarkVS/nopack-gopb-unmarshal-10          	  593708	      1849 ns/op	    2104 B/op	      44 allocs/op
BenchmarkVS/map-int-google-marshal-10         	  136328	      8729 ns/op	    5328 B/op	     137 allocs/op
BenchmarkVS/map-int-gopb-marshal-10           	  763930	      1522 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/map-int-google-unmarshal-10       	  176167	      7034 ns/op	    4680 B/op	     114 allocs/op
BenchmarkVS/map-int-gopb-unmarshal-10         	  456044	      2490 ns/op	    4040 B/op	      38 allocs/op
BenchmarkVS/map-all-google-marshal-10         	   96744	     12281 ns/op	    5712 B/op	     215 allocs/op
BenchmarkVS/map-all-gopb-marshal-10           	  575503	      2013 ns/op	     640 B/op	       1 allocs/op
BenchmarkVS/map-all-google-unmarshal-10       	  103428	     11695 ns/op	    4888 B/op	     179 allocs/op
BenchmarkVS/map-all-gopb-unmarshal-10         	  371188	      3055 ns/op	    3800 B/op	      36 allocs/op
PASS
ok  	github.com/aggronmagi/benchpb	34.338s
```

与gogo/prorobuf相比

``` shell
➜  gogofaster git:(main) go test -bench=. -benchmem
goos: darwin
goarch: amd64
pkg: github.com/aggronmagi/benchpb/gogofaster
cpu: VirtualApple @ 2.50GHz
BenchmarkVS/complex-gogofaster-marshal-10         	  220010	      4675 ns/op	    2376 B/op	       8 allocs/op
BenchmarkVS/complex-gopb-marshal-10               	  253090	      4072 ns/op	    2048 B/op	       1 allocs/op
BenchmarkVS/complex-gogofaster-unmarshal-10       	  161184	      7109 ns/op	    8000 B/op	     150 allocs/op
BenchmarkVS/complex-gopb-unmarshal-10             	  135082	      7820 ns/op	    8152 B/op	     151 allocs/op
BenchmarkVS/basic_type-gogofaster-marshal-10      	 7020572	       183.1 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-gopb-marshal-10            	 7500072	       157.8 ns/op	      96 B/op	       1 allocs/op
BenchmarkVS/basic_type-gogofaster-unmarshal-10    	 2423949	       504.6 ns/op	    1304 B/op	       4 allocs/op
BenchmarkVS/basic_type-gopb-unmarshal-10          	 2634120	       455.3 ns/op	    1304 B/op	       4 allocs/op
BenchmarkVS/repeated-gogofaster-marshal-10        	 1942646	       556.4 ns/op	     680 B/op	       8 allocs/op
BenchmarkVS/repeated-gopb-marshal-10              	 3070996	       384.9 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/repeated-gogofaster-unmarshal-10      	  963336	      1242 ns/op	    1848 B/op	      28 allocs/op
BenchmarkVS/repeated-gopb-unmarshal-10            	  939012	      1265 ns/op	    2024 B/op	      33 allocs/op
BenchmarkVS/nopack-gogofaster-marshal-10          	 2694494	       447.0 ns/op	     416 B/op	       1 allocs/op
BenchmarkVS/nopack-gopb-marshal-10                	 3267100	       364.2 ns/op	     416 B/op	       1 allocs/op
BenchmarkVS/nopack-gogofaster-unmarshal-10        	  678048	      1725 ns/op	    2136 B/op	      48 allocs/op
BenchmarkVS/nopack-gopb-unmarshal-10              	  686800	      1738 ns/op	    2104 B/op	      44 allocs/op
BenchmarkVS/map-int-gogofaster-marshal-10         	  734241	      1610 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/map-int-gopb-marshal-10               	  804480	      1517 ns/op	     352 B/op	       1 allocs/op
BenchmarkVS/map-int-gogofaster-unmarshal-10       	  540970	      2165 ns/op	    4040 B/op	      38 allocs/op
BenchmarkVS/map-int-gopb-unmarshal-10             	  521365	      2253 ns/op	    4040 B/op	      38 allocs/op
BenchmarkVS/map-all-gogofaster-marshal-10         	  536992	      2204 ns/op	     640 B/op	       1 allocs/op
BenchmarkVS/map-all-gopb-marshal-10               	  598638	      1971 ns/op	     640 B/op	       1 allocs/op
BenchmarkVS/map-all-gogofaster-unmarshal-10       	  428917	      2603 ns/op	    3800 B/op	      36 allocs/op
BenchmarkVS/map-all-gopb-unmarshal-10             	  405924	      2840 ns/op	    3800 B/op	      36 allocs/op
PASS
ok  	github.com/aggronmagi/benchpb/gogofaster	32.544s
```

## 定制
### 自定义结构实现 protobuf 序列化能力. 
填充 gengo/GenerateStruct ,然后调用以下函数来生成. 需要参照现有代码,处理import包问题.
``` go
func GenExec(data *GenerateStruct) (_ []byte, err error)
```
### 调整生成代码. 
修改 gengo/gen_template.go 中的模板,调整生成代码. 
### 添加自定义生成. 
参照 genparse/gen_zap.go 
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


