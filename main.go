package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go.uber.org/multierr"
	"google.golang.org/protobuf/compiler/protogen"
)

const (
	genGoDocURL     = "https://github.com/aggronmagi/protoc-gen-gopb"
	genProtoWirePkg = "google.golang.org/protobuf/encoding/protowire"
)

var (
	Version = "0.0.1"
)

// 命令行配置
var cfg = struct {
	zap     bool
	get     bool
	debug   bool
	wirepkg string
}{
	zap:     true,
	get:     false,
	debug:   true,
	wirepkg: genProtoWirePkg,
}

func setupConfigOption(flags *flag.FlagSet) {
	// 如果环境变量设置了值, 读取作为为默认值. 优先使用传递的参数
	env := os.Getenv("GOPB_WIRE_PACKAGE")
	if env != "" {
		cfg.wirepkg = env
	}
	env = os.Getenv("GOPB_GEN_GET")
	if env != "" {
		cfg.get, _ = strconv.ParseBool(env)
	}
	env = os.Getenv("GOPB_GEN_ZAP")
	if env != "" {
		cfg.zap, _ = strconv.ParseBool(env)
	}
	env = os.Getenv("GOPB_GEN_DEBUG")
	if env != "" {
		cfg.debug, _ = strconv.ParseBool(env)
	}

	flags.BoolVar(&cfg.zap, "zap", cfg.zap, "generate zap log interface")
	flags.BoolVar(&cfg.get, "get", cfg.get, "generate message getter method")
	flags.StringVar(&cfg.wirepkg, "pbwire", cfg.wirepkg, "use protobuf wire package")
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintf(os.Stdout, "%v %v\n", filepath.Base(os.Args[0]), Version)
		os.Exit(0)
	}
	if len(os.Args) == 2 && os.Args[1] == "--help" {
		fmt.Fprintf(os.Stdout, "See "+genGoDocURL+" for usage information.\n")
		os.Exit(0)
	}
	var flags flag.FlagSet
	plugins := flags.String("plugins", "", "deprecated option")
	setupConfigOption(&flags)

	protogen.Options{
		ParamFunc: flags.Set,
	}.Run(func(gen *protogen.Plugin) (err error) {
		if *plugins != "" {
			return errors.New("protoc-gen-gopb: plugins are not supported; ")
		}
		// 生成消息
		for _, f := range gen.Files {
			if !f.Generate {
				continue
			}
			err = multierr.Append(err, genProtobuf(gen, f))
		}
		//gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		return
	})
}

func genProtobuf(gen *protogen.Plugin, f *protogen.File) (err error) {
	filename := f.GeneratedFilenamePrefix + ".gopb.go"
	//log.Println(filename)
	g := gen.NewGeneratedFile(filename, f.GoImportPath)
	genStandaloneComments(g, f, int32(FileDescriptorProto_Syntax_field_number))
	genGeneratedHeader(gen, g, f)
	genStandaloneComments(g, f, int32(FileDescriptorProto_Package_field_number))

	g.P("package ", f.GoPackageName)
	g.P()
	for _, e := range f.Enums {
		err = multierr.Append(err, genEnum(g, f, e))
	}
	for _, m := range f.Messages {
		err = multierr.Append(err, genMessage(g, f, m))
	}
	return
}
