package mvgen

import (
	"context"
	"io"
	"log"
	"reflect"
	"strings"

	"github.com/bufbuild/protocompile"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	pbBuilder "github.com/jhump/protoreflect/desc/builder"

	"github.com/jhump/protoreflect/desc/protoprint"
	"github.com/mainvec/ugo/omap"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

//NOTE: where possible, we are using the latest 'google.golang.org/protobuf' and
// not older api 'github.com/golang/protobuf'. we still need github.com/jhump/protoreflect
// which requires the older api, mainly for the desc.builder.

// Build a Protobuff3 schema from WO JSON Schema
func BuildProtoBuffDefFromJSON(srvDefJsonReader io.Reader) (protoreflect.FileDescriptor, error) {
	srvDef, err := BuildSrvDefFromJSON(srvDefJsonReader)
	if err != nil {
		return nil, err
	}

	return BuildProtoBuffDefFromSrvDef(srvDef)
}

// Build a Protobuff3 schema from WO SrvDef
func BuildProtoBuffDefFromSrvDef(srvDef *SrvDef) (protoreflect.FileDescriptor, error) {

	name := srvDef.Name
	namespace := srvDef.Namespace
	b := pbBuilder.NewFile(name)
	b.SetComments(buildComments(srvDef.Desc))
	b.SetPackageName(namespace + "." + name)
	b.SetProto3(true)
	/*
		processOptions(b, jsonSpec)
		//build recordDef messsgaes
		b.SetProto3(true)




	*/
	processOptions(b, srvDef)
	processRecordDefMessages(b, srvDef)
	processCommandMessages(b, srvDef)
	processImports(b, srvDef)
	pb3desc, err := b.Build()
	if err != nil {
		return nil, err
	}
	if debug := false; debug {

		printer := &protoprint.Printer{}
		descString, err := printer.PrintProtoToString(pb3desc)
		if err == nil {
			log.Println(descString)
		}
	}
	return pb3desc.UnwrapFile(), nil
}

func ParseProto3Definition(name string, pb3Definition []byte) (protoreflect.FileDescriptor, error) {

	// parser := &protoparse.Parser{}
	// parser.Accessor = protoparse.FileContentsFromMap(map[string]string{
	// 	name: string(pb3Definition),
	// })
	// parser.IncludeSourceCodeInfo = false
	// //for now only the first one. don't know how to pass var len arg
	// descrption, err := parser.ParseFiles(name)
	// if err != nil {
	// 	return nil, err
	// }
	// return descrption[0].UnwrapFile(), nil
	//If you used protoparse.FileContentsFromMap,
	//in this new repo you'll use a protocompile.SourceResolver and then use protocompile.SourceAccessorFromMap as its accessor function.
	//
	resolver := &protocompile.SourceResolver{
		Accessor: protocompile.SourceAccessorFromMap(map[string]string{
			name: string(pb3Definition),
		}),
	}
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(resolver),
	}
	files, err := compiler.Compile(context.Background(), name)
	if err != nil {
		return nil, err
	}
	return files[0], nil

}

func processOptions(b *pbBuilder.FileBuilder, srvDef *SrvDef) {
	options := srvDef.GenOpts
	fileOptions := &dpb.FileOptions{}
	gopkg, ok := options["go_api_package"]
	if ok {
		fileOptions.GoPackage = &gopkg
	} else {
		gopkg, ok = options["go_package"]
		if ok {
			fileOptions.GoPackage = &gopkg
		}
	}

	for name, _ := range options {

		switch name {
		case "go_package":
		case "go_api_package":
			//OK
		default:
			log.Fatalf("unsupported gen_option [%v]", name)
		}

	}
	//Always set .proto go package if not provided
	if fileOptions.GoPackage == nil {
		defaultGOAPIPackage := srvDef.Base + "/" + srvDef.Name + "/go/" + srvDef.Name + "api"
		fileOptions.GoPackage = &defaultGOAPIPackage
	}

	b.SetOptions(fileOptions)
	//Add missing options:

}

func processImports(b *pbBuilder.FileBuilder, srvDef *SrvDef) {
	_ = srvDef
	extDeps := []string{
		"google/protobuf/timestamp.proto",
		"google/protobuf/duration.proto",
	}
	for _, v := range extDeps {
		fd, err := desc.LoadFileDescriptor(v)
		if err != nil {
			log.Fatalf("loading dep failed")
		}

		fdesp, err := pbBuilder.FromFile(fd)
		if err != nil {
			log.Fatalf("loading dep failed")
		}
		b.AddDependency(fdesp)
	}

}

func processRecordDefMessages(b *pbBuilder.FileBuilder, srvDef *SrvDef) {
	recordDefs := srvDef.Records
	// //FIXME should process dependent recrods first and detect cycles
	// recs:=make([]string,0,len(recordDefs))

	// for r,recDef  := range recordDefs {
	// 	recs=prepareRecordsDepdencies(recs,r,recDef)
	// }
	// mapForEachKeySorted(recordDefs, func(recname string, recordDef RecordDef) any {
	// 	mb := pbBuilder.NewMessage(recname)
	// 	processMessageFields(srvDef, b, mb, recordDef.Fields)
	// 	b.AddMessage(mb)
	// 	return nil
	// })
	for recname, recordDef := range recordDefs {
		processRecordDef(b, srvDef, recname, recordDef)
	}
}

func processRecordDef(b *pbBuilder.FileBuilder, srvDef *SrvDef, recname string, recordDef RecordDef) {
	//already processed via dependencies
	if b.GetMessage(recname) != nil {
		return
	}
	mb := pbBuilder.NewMessage(recname)
	processMessageFields(srvDef, b, mb, recordDef.Fields)
	b.AddMessage(mb)
}

func processCommandMessages(b *pbBuilder.FileBuilder, srvDef *SrvDef) {
	commands := srvDef.Commands
	mapForEachKeySorted(commands, func(commandName string, commandDef CommandDef) any {
		//name := commandDef.Id
		mb := pbBuilder.NewMessage(commandName)
		processMessageFields(srvDef, b, mb, commandDef.Fields)
		b.AddMessage(mb)

		//process command result, an empty one of no result fields
		resultName := commandName + "Result"
		mbresult := pbBuilder.NewMessage(resultName)
		processMessageFields(srvDef, b, mbresult, commandDef.ResultFields)
		// _, ok := commandDef["resultFields"].(map[string]interface{})
		// if ok {
		// 	processMessageFields(jsonSpec, b, mbresult, commandDef, "resultFields")
		// }
		b.AddMessage(mbresult)
		//pbBuilder.NewMessage(mbresult)
		return nil
	})
}

func processMessageFields(srvDef *SrvDef, b *pbBuilder.FileBuilder, mb *pbBuilder.MessageBuilder, fieldDefs FieldDefs) {
	//order by fnum
	lessFunc := func(i, j FieldDef) bool {
		return i.Fnum < j.Fnum
	}
	for iter := omap.IterateByValue(fieldDefs, lessFunc); iter.HasNext(); {
		name, fieldDef := iter.Next()
		ftype := fieldDef.Type

		//not the best way, need refactor
		switch ftype {
		case "oneOf":
			//Special sace for oneOf
			processOneField(srvDef, b, mb, name, &fieldDef)
		default:
			processMessageField(srvDef, b, mb, name, &fieldDef)
		}
	}

}

func processField(srvDef *SrvDef, b *pbBuilder.FileBuilder, fname string, fieldDef *FieldDef) (*pbBuilder.FieldBuilder, error) {
	ftype := fieldDef.Type

	var fbuilder *pbBuilder.FieldBuilder
	switch ftype {
	case "string":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeString())
	case "boolean":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeBool())
	case "int32":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeInt32())
	case "int64":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeInt64())
	case "float":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeFloat())
	case "double":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeDouble())
	case "bytes":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeBytes())
	case "uint32":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeUInt32())
	case "sint32":
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeSInt32())
	case "timestamp":
		msgDes, err := desc.LoadMessageDescriptor("google.protobuf.Timestamp")
		if err != nil {
			log.Fatalf("could load timestamp message for field [%v]", fname)
		}
		msgB, err := pbBuilder.FromMessage(msgDes)
		if err != nil {
			log.Fatalf("could prepare msg builder for timestamp message for field [%v]", fname)
		}
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeMessage(msgB))
	case "duration":
		msgDes, err := desc.LoadMessageDescriptor("google.protobuf.Duration")
		if err != nil {
			log.Fatalf("could load timestamp message for field [%v]", fname)
		}
		msgB, err := pbBuilder.FromMessage(msgDes)
		if err != nil {
			log.Fatalf("could prepare msg builder for timestamp message for field [%v]", fname)
		}
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeMessage(msgB))

	case "map":
		fbuilder = pbBuilder.NewMapField(fname, pbBuilder.FieldTypeString(), pbBuilder.FieldTypeString())
	case "recRef":
		//TODO: could not figure a way to enforce this
		// in th escheam. hard coding the lookup until a better
		//way is found.
		//For now it should start with #/recordsDefs/

		ref := fieldDef.RecRef

		prefix := "#/recordsDefs/"
		ok := strings.HasPrefix(ref, prefix)
		if !ok {
			log.Fatalf("not a valid $ref for field [%v]", fname)
		}
		recordDefName := ref[len(prefix):]

		refmb := b.GetMessage(recordDefName)
		if refmb == nil {
			//FIXME: should detect cycles and fail
			recordDef, ok := srvDef.Records[recordDefName]
			if !ok {
				log.Fatalf("cannot find rerodDefname [%v] for field [%v]. ", recordDefName, fname)
			}
			processRecordDef(b, srvDef, recordDefName, recordDef)
			refmb = b.GetMessage(recordDefName)
			if refmb == nil {
				log.Fatalf("cannot find rerodDefname [%v] for field [%v]. ", recordDefName, fname)
			}
		}
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeMessage(refmb))
	case "record":
		//log.Printf("TODO record fields [%v]", fname)
		recordDefName := "" //TODO fieldDef.RecName

		refmb := b.GetMessage(recordDefName)
		if refmb == nil {
			log.Fatalf("cannot find rerodDefname [%v] for field [%v]. ", recordDefName, fname)
		}
		fbuilder = pbBuilder.NewField(fname, pbBuilder.FieldTypeMessage(refmb))
	default:
		log.Fatalf("not a valid field type for field [%v]", fname)
	}
	fnum := fieldDef.Fnum
	if fnum < 1 {
		log.Fatalf("not a valid fnum[%v] for field [%v]. should be > 0", fnum, fname)
	}
	fbuilder.SetNumber(int32(fnum))

	//note: fbuilder.isRepeated used to cater for Map field builder
	repeated := fieldDef.Repeated || fbuilder.IsRepeated()
	//important to set optional before repeat below.
	fbuilder.SetOptional()
	if repeated {
		fbuilder.SetRepeated()
	}

	return fbuilder, nil
}

func processOneField(srvDef *SrvDef, b *pbBuilder.FileBuilder, mb *pbBuilder.MessageBuilder, fname string, fieldDef *FieldDef) {
	_ = fieldDef
	fieldDefs := FieldDefs{} //TODO fieldDef["fields"].(map[string]interface{})

	oneOfBuilder := pbBuilder.NewOneOf(fname)

	for name, fieldDef := range fieldDefs {

		ftype := fieldDef.Type

		//not the best way, need refactor
		switch ftype {
		case "oneOf":
			log.Fatalf("nested OneOf not yet supported: [%v]", name)
		default:
			fbuilder, err := processField(srvDef, b, name, &fieldDef)
			if err != nil {
				log.Fatal(err)
			}
			oneOfBuilder.AddChoice(fbuilder)
		}

	}
	mb.AddOneOf(oneOfBuilder)
}

func processMessageField(srvDef *SrvDef, b *pbBuilder.FileBuilder, mb *pbBuilder.MessageBuilder, fname string, fieldDef *FieldDef) {
	fbuilder, err := processField(srvDef, b, fname, fieldDef)
	if err != nil {
		log.Fatal(err)
	}
	mb.AddField(fbuilder)
}

func buildComments(comments string) pbBuilder.Comments {
	return pbBuilder.Comments{
		LeadingComment: comments,
	}
}

func RemoveSourceCodeInfo(desc *desc.FileDescriptor) *desc.FileDescriptor {
	desc.AsFileDescriptorProto().SourceCodeInfo = nil
	return desc
}

func GenerateProtobuf3FromFileDesc(pbFielDesc protoreflect.FileDescriptor, w io.Writer) error {
	printer := &protoprint.Printer{}
	olddesc, err := desc.WrapFile(pbFielDesc)
	if err != nil {
		return err
	}

	err = printer.PrintProtoFile(olddesc, w)
	if err != nil {
		return err
	}
	// _, err = io.WriteString(w, descString)
	// if err != nil {
	// 	return err
	// }
	return nil
}

func GenerateProtobuf3FromSrvDef(srvDef *SrvDef, w io.Writer) error {
	pbFielDesc, err := BuildProtoBuffDefFromSrvDef(srvDef)
	if err != nil {
		return err
	}
	return GenerateProtobuf3FromFileDesc(pbFielDesc, w)
}

func FileDescriptorEqual(desc1, desc2 protoreflect.FileDescriptor) bool {
	//Ignore dependencies for now. deep equal not working on them
	//dep1 := desc1.GetDependencies()
	//dep2 := desc2.GetDependencies()
	//not woring this: equal := cmp.Equal(desc1, desc2)
	s1 := desc1.Syntax()
	s2 := desc2.Syntax()
	// n1 := desc1.Name()
	// n2 := desc2.Name()
	nf1 := desc1.FullName()
	nf2 := desc2.FullName()
	p1 := desc1.Package()
	p2 := desc2.Package()
	equal := s1 == s2 && p1 == p2 && nf1 == nf2

	// equal :=
	// 	(desc1.Syntax() == desc2.Syntax()) &&
	// 		(desc1.Name() == desc2.Name()) &&
	// 		(desc1.Package() == desc2.Package())
	equal = equal && isMessagesEqual(desc1.Messages(), desc2.Messages())
	//TODO Rethink how to default gen options
	//equal = equal && isOptionsEqual(desc1.GetFileOptions(), desc2.GetFileOptions())

	return equal
}

func isOptionsEqual(op1, op2 *descriptorpb.FileOptions) bool {
	return reflect.DeepEqual(op1, op2)
}

func isMessagesEqual(a, b protoreflect.MessageDescriptors) bool {
	//order by message name first
	eq := true
	if a.Len() != b.Len() {
		return false
	}
	//FIXME very fragile comparision
	for i := 0; i < a.Len(); i++ {
		aM := a.Get(i)
		if !isMessageEqual(aM, b.ByName(aM.Name())) {
			return false
		}
	}
	return eq
}

func isMessageEqual(a, b protoreflect.MessageDescriptor) bool {

	if a.FullName() != b.FullName() {
		return false
	}
	if !isFieldsEqual(a.Fields(), b.Fields()) {
		return false
	}
	return true
}

func isFieldsEqual(a, b protoreflect.FieldDescriptors) bool {
	//order by field number first
	//	sortFun := func(i, j protoreflect.FieldDescriptor) bool { return i.Number() < j.Number() }
	//	sortedA := sortBy(a, sortFun)
	//	sortedB := sortBy(b, sortFun)

	eq := true
	if a.Len() != b.Len() {
		return false
	}
	//FIXME very fragile comparision
	for i := 0; i < a.Len(); i++ {
		aF := a.Get(i)
		if !isFieldEqual(aF, b.ByNumber(aF.Number())) {
			return false
		}
	}
	return eq
}

func isFieldEqual(a, b protoreflect.FieldDescriptor) bool {

	if a.FullName() != b.FullName() {
		return false
	}
	aName := a.Name()
	bName := b.Name()

	aNum := a.Number()
	bNum := b.Number()

	aKind := a.Kind()
	bKind := b.Kind()

	eq := aName == bName && aNum == bNum && aKind == bKind
	//eq := a.Name() == b.Name() && a.Number() == b.Number() && a.Kind() == b.Kind()
	if a.IsMap() || b.IsMap() {
		//for some reason 'optional' lable is not set on map entry
		//key value, use type comparsion for now
		eq = eq && (a.MapKey().Kind() == b.MapKey().Kind() && a.MapValue().Kind() == b.MapValue().Kind())
	}
	if !eq {
		return false
	}
	return eq
}
