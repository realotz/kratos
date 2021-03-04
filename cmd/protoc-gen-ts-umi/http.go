package main

import (
	"fmt"
	"google.golang.org/protobuf/reflect/protoreflect"
	"strings"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

var methodSets = make(map[string]int)

type serviceDesc struct {
	ServiceType string // Greeter
	Comments    string // Comments
	ServiceName string // helloworld.Greeter
	Metadata    string // api/helloworld/helloworld.proto
	Methods     []*methodDesc
	MethodSets  map[string]*methodDesc
}

type methodDesc struct {
	// method
	Name    string
	Num     int
	Vars    []string
	Forms   []string
	Request string
	Reply   string
	// http_rule
	Path         string
	Method       string
	Body         string
	ResponseBody string
}

// generateFile generates a _http.pb.go file containing kratos errors definitions.
func generateFile(gen *protogen.Plugin, file *protogen.File) *protogen.GeneratedFile {
	if len(file.Services) == 0 {
		return nil
	}
	filename := file.GeneratedFilenamePrefix + "_service.pb.ts"
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	g.P("// @ts-ignore")
	g.P("/* eslint-disable */")
	g.P("// Code generated by protoc-gen-ts-umi. DO NOT EDIT.")
	g.P("import {request} from 'umi';")
	g.P()

	g.P(fmt.Sprintf("const APIService = '/api';"))
	generateFileContent(gen, file, g, true)

	messageService := file.GeneratedFilenamePrefix + ".d.ts"
	g = gen.NewGeneratedFile(messageService, file.GoImportPath)
	g.P("// @ts-ignore")
	g.P("/* eslint-disable */")
	g.P("// Code generated by protoc-gen-ts-umi. DO NOT EDIT.")
	g.P()
	generateFileContent(gen, file, g, false)
	return g
}

// generateFileContent generates the kratos errors definitions, excluding the package statement.
func generateFileContent(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service bool) {
	if len(file.Services) == 0 {
		return
	}
	g.P("// This is a compile-time assertion to ensure that this generated file")
	g.P("// is compatible with the kratos package it is being compiled against.")
	g.P()
	if service {
		g.P("type Options = {\n  [key: string]: any\n}")
		g.P()
		for _, service := range file.Services {
			genService(gen, file, g, service)
		}
	} else {
		messagesChan := make(chan protoreflect.MessageDescriptor, len(file.Messages)*10)
		for _, message := range file.Messages {
			messagesChan <- message.Desc
		}
		hash := make(map[protoreflect.FullName]map[protoreflect.Name]protoreflect.MessageDescriptor)
		for len(messagesChan) > 0 {
			getMessage(<-messagesChan, messagesChan, hash)
		}
		for k, v := range hash {
			g.P(fmt.Sprintf("declare namespace %s {", k))
			for _, m := range v {
				genMessage(file, g, m)
			}
			g.P(fmt.Sprintf("}"))
			g.P()
		}

	}

}

func Marshal(fullname protoreflect.FullName) protoreflect.FullName {
	name := string(fullname)
	if name == "" {
		return ""
	}
	temp := strings.Split(name, ".")
	var s string
	for _, v := range temp {
		vv := []rune(v)
		if len(vv) > 0 {
			if bool(vv[0] >= 'a' && vv[0] <= 'z') { //首字母大写
				vv[0] -= 32
			}
			s += string(vv)
		}
	}
	return protoreflect.FullName(s)
}

func getMessage(message protoreflect.MessageDescriptor,
	ch chan protoreflect.MessageDescriptor, hash map[protoreflect.FullName]map[protoreflect.Name]protoreflect.MessageDescriptor) {
	if _, ok := hash[Marshal(message.ParentFile().Package())]; ok {
		if _, ok := hash[Marshal(message.ParentFile().Package())][message.Name()]; ok {
			return
		}
	} else {
		hash[Marshal(message.ParentFile().Package())] = make(map[protoreflect.Name]protoreflect.MessageDescriptor)
	}
	hash[Marshal(message.ParentFile().Package())][message.Name()] = message
	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.Kind() == protoreflect.MessageKind {
			if _, ok := hash[message.ParentFile().Package()][field.Message().Name()]; !ok {
				ch <- field.Message()
			}
		}
	}
}

func genMessage(file *protogen.File, g *protogen.GeneratedFile, message protoreflect.MessageDescriptor) {
	g.P(fmt.Sprintf("	/** %s */", message.Name()))
	g.P(fmt.Sprintf("	type %s = {", message.Name()))
	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		g.P(fmt.Sprintf("		%s?:%s", field.Name(), messageKindType(file, field)))
	}
	g.P(fmt.Sprintf("	}"))
}

func messageKindType(file *protogen.File, desc protoreflect.FieldDescriptor) string {
	if desc.IsMap() {
		return fmt.Sprintf("Map<%s,%s>", messageKindType(file, desc.MapKey()), messageKindType(file, desc.MapValue()))
	}
	switch desc.Kind() {
	case protoreflect.BoolKind:
		if desc.IsList() {
			return "Array<boolean>"
		}
		return "boolean"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Uint64Kind,
		protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		if desc.IsList() {
			return "Array<number>"
		}
		return "number"
	case protoreflect.MessageKind:
		if desc.IsList() {
			return fmt.Sprintf("Array<%s.%s>", Marshal(desc.Message().ParentFile().Package()), string(desc.Message().Name()))
		}
		return fmt.Sprintf("%s.%s", Marshal(desc.Message().ParentFile().Package()), string(desc.Message().Name()))
	case protoreflect.EnumKind:
		return "Array<any>"
	default:
		if desc.IsList() {
			return "Array<string>"
		}
		return "string"
	}
}

func genService(gen *protogen.Plugin, file *protogen.File, g *protogen.GeneratedFile, service *protogen.Service) {
	if service.Desc.Options().(*descriptorpb.ServiceOptions).GetDeprecated() {
		g.P("//")
		g.P(deprecationComment)
	}
	// HTTP Server.
	sd := &serviceDesc{
		ServiceType: service.GoName,
		ServiceName: string(service.Desc.FullName()),
		Metadata:    file.Desc.Path(),
	}
	for _, method := range service.Methods {
		rule, ok := proto.GetExtension(method.Desc.Options(), annotations.E_Http).(*annotations.HttpRule)
		if rule != nil && ok {
			for _, bind := range rule.AdditionalBindings {
				sd.Methods = append(sd.Methods, buildHTTPRule(method, bind))
			}
			sd.Methods = append(sd.Methods, buildHTTPRule(method, rule))
		} else {
			path := fmt.Sprintf("/%s/%s", service.Desc.FullName(), method.Desc.Name())
			sd.Methods = append(sd.Methods, buildMethodDesc(method, "POST", path))
		}
	}
	for k, method := range sd.Methods {
		c := service.Methods[k].Comments.Leading.String()
		c = strings.ReplaceAll(c, "//", "")
		c = strings.ReplaceAll(c, "\n", "")

		g.P(fmt.Sprintf("/** %s %s /api */", method.Name, c))
		g.P(fmt.Sprintf("export async function %s(params: %s.%s, options?: Options) {", method.Name, Marshal(file.Desc.Package()), method.Request))
		g.P(fmt.Sprintf("	return request<%s.%s>(APIService + '%s', {", Marshal(file.Desc.Package()), method.Reply, method.Path))
		g.P(fmt.Sprintf("    	method: '%s',", method.Method))
		if method.Body != "" {
			g.P(fmt.Sprintf("		headers: {'Content-Type': 'application/json',},"))
			g.P(fmt.Sprintf("    	data: {...params},"))
		} else {

			g.P(fmt.Sprintf("    	params: {...params},"))
		}
		g.P(fmt.Sprintf("    	...(options || {}),"))
		g.P(fmt.Sprintf("	});"))
		g.P(fmt.Sprintf("}"))
		g.P()
	}
}

func buildHTTPRule(m *protogen.Method, rule *annotations.HttpRule) *methodDesc {
	var (
		path         string
		method       string
		body         string
		responseBody string
	)
	switch pattern := rule.Pattern.(type) {
	case *annotations.HttpRule_Get:
		path = pattern.Get
		method = "GET"
	case *annotations.HttpRule_Put:
		path = pattern.Put
		method = "PUT"
	case *annotations.HttpRule_Post:
		path = pattern.Post
		method = "POST"
	case *annotations.HttpRule_Delete:
		path = pattern.Delete
		method = "DELETE"
	case *annotations.HttpRule_Patch:
		path = pattern.Patch
		method = "PATCH"
	case *annotations.HttpRule_Custom:
		path = pattern.Custom.Path
		method = pattern.Custom.Kind
	}
	body = rule.Body
	responseBody = rule.ResponseBody
	md := buildMethodDesc(m, method, path)
	if body != "" {
		md.Body = "." + camelCaseVars(body)
	}
	if responseBody != "" {
		md.ResponseBody = "." + camelCaseVars(responseBody)
	}
	return md
}

func buildMethodDesc(m *protogen.Method, method, path string) *methodDesc {
	defer func() { methodSets[m.GoName]++ }()
	return &methodDesc{
		Name:    m.GoName,
		Num:     methodSets[m.GoName],
		Request: m.Input.GoIdent.GoName,
		Reply:   m.Output.GoIdent.GoName,
		Path:    path,
		Method:  method,
		Vars:    buildPathVars(m, path),
	}
}

func buildPathVars(method *protogen.Method, path string) (res []string) {
	for _, v := range strings.Split(path, "/") {
		if strings.HasPrefix(v, "{") && strings.HasSuffix(v, "}") {
			name := strings.TrimRight(strings.TrimLeft(v, "{"), "}")
			res = append(res, name)
		}
	}
	return
}

func camelCaseVars(s string) string {
	var (
		vars []string
		subs = strings.Split(s, ".")
	)
	for _, sub := range subs {
		vars = append(vars, camelCase(sub))
	}
	return strings.Join(vars, ".")
}

// camelCase returns the CamelCased name.
// If there is an interior underscore followed by a lower case letter,
// drop the underscore and convert the letter to upper case.
// There is a remote possibility of this rewrite causing a name collision,
// but it's so remote we're prepared to pretend it's nonexistent - since the
// C++ generator lowercases names, it's extremely unlikely to have two fields
// with different capitalizations.
// In short, _my_field_name_2 becomes XMyFieldName_2.
func camelCase(s string) string {
	if s == "" {
		return ""
	}
	t := make([]byte, 0, 32)
	i := 0
	if s[0] == '_' {
		// Need a capital letter; drop the '_'.
		t = append(t, 'X')
		i++
	}
	// Invariant: if the next letter is lower case, it must be converted
	// to upper case.
	// That is, we process a word at a time, where words are marked by _ or
	// upper case letter. Digits are treated as words.
	for ; i < len(s); i++ {
		c := s[i]
		if c == '_' && i+1 < len(s) && isASCIILower(s[i+1]) {
			continue // Skip the underscore in s.
		}
		if isASCIIDigit(c) {
			t = append(t, c)
			continue
		}
		// Assume we have a letter now - if not, it's a bogus identifier.
		// The next word is a sequence of characters that must start upper case.
		if isASCIILower(c) {
			c ^= ' ' // Make it a capital letter.
		}
		t = append(t, c) // Guaranteed not lower case.
		// Accept lower case sequence that follows.
		for i+1 < len(s) && isASCIILower(s[i+1]) {
			i++
			t = append(t, s[i])
		}
	}
	return string(t)
}

// Is c an ASCII lower-case letter?
func isASCIILower(c byte) bool {
	return 'a' <= c && c <= 'z'
}

// Is c an ASCII digit?
func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

const deprecationComment = "// Deprecated: Do not use."
