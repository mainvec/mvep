package wop

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/xeipuuv/gojsonschema"
	"golang.org/x/exp/constraints"
)

//go:embed resources
var resources embed.FS

type GenOpt struct {
	OptName  string
	OptValue string
}

//type CommandId string

type CommandDef struct {
	Id     string    `json:"-"`
	Title  string    `json:"title,omitempty"`
	Desc   string    `json:"description,omitempty"`
	Fields FieldDefs `json:"fields,omitempty"`
}

type RecordDef struct {
	Name   string    `json:"name"`
	Title  string    `json:"title,omitempty"`
	Desc   string    `json:"description,omitempty"`
	Fields FieldDefs `json:"fields,omitempty"`
}

type FieldDataType string

/*
		"string",
                "boolean",
                "int32",
                "int64",
                "float",
                "double",
                "bytes",
                "uint32",
                "sint32",
                "timestamp",
                "duration"
*/
const (
	STRING  FieldDataType = "string"
	BOOLEAN FieldDataType = "boolean"
	INT32   FieldDataType = "int32"
	RECREF  FieldDataType = "recRef"
	MAP     FieldDataType = "map"
)

type FieldDef struct {
	Id           string        `json:"-"`
	Title        string        `json:"title,omitempty"`
	Desc         string        `json:"description,omitempty"`
	Fnum         int32         `json:"fnum"`
	Type         FieldDataType `json:"type"`
	Repeated     bool          `json:"repeated,omitempty"`
	RecRef       string        `json:"$ref,omitempty"`
	MapValueType string        `json:"valueType,omitempty"`
}

type OMap[K constraints.Ordered, V any] map[K]V

type iterator[K constraints.Ordered, V any] struct {
	i     int
	_keys []K
	_map  map[K]V
}

func (iter *iterator[K, V]) HasNext() bool {
	return int(iter.i) < len(iter._keys)
}

func (iter *iterator[K, V]) Next() (K, V) {
	if !iter.HasNext() {
		panic("iteration has no more elements, shold us HasNext")
	}
	k := iter._keys[iter.i]
	iter.i++
	return k, iter._map[k]

}

// func IteratorByKey[K constraints.Ordered, V any](_omap interface{}) iterator[K, V] {
// 	omap, ok := _omap.(OMap[K, V])
// 	if !ok {
// 		panic("iterationByKey needs OMap")
// 	}
// 	return iteratorByKey(omap)
// }

func IteratorByKey[K constraints.Ordered, V any](omap map[K]V) iterator[K, V] {

	keys := make([]K, 0, len(omap))
	for k := range omap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return iterator[K, V]{
		i:     0,
		_keys: keys,
		_map:  omap,
	}
}

func (omap OMap[K, V]) IterateByKey() iterator[K, V] {
	return IteratorByKey(omap)
}

type mapItem[K constraints.Ordered, V any] struct {
	_k K
	_v V
}

//Create a iterator for the map, ordered by map vlaues, using the lessFunc
func IterateByValue[K constraints.Ordered, V any](omap map[K]V, lessFunc func(i, j V) bool) iterator[K, V] {

	items := make([]*mapItem[K, V], 0, len(omap))
	for k, v := range omap {
		items = append(items, &mapItem[K, V]{
			_k: k,
			_v: v,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return lessFunc(items[i]._v, items[j]._v)
	})

	keys := make([]K, 0, len(omap))
	for _, item := range items {
		keys = append(keys, item._k)
	}
	return iterator[K, V]{
		i:     0,
		_keys: keys,
		_map:  omap,
	}
}

func (omap OMap[K, V]) IterateByValue(lessFunc func(i, j V) bool) iterator[K, V] {
	//Can only be used by direct OMap. Cannot be used by structs "inherting" OMap
	return IterateByValue(omap, lessFunc)
}

type CommandDefs OMap[string, CommandDef]
type FieldDefs OMap[string, FieldDef]
type RecordsDefs OMap[string, RecordDef]
type GenOptsDef OMap[string, string]

type SrvDef struct {
	Id        string      `json:"$id"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Base      string      `json:"base,omitempty"`
	Desc      string      `json:"description,omitempty"`
	Commands  CommandDefs `json:"commands,omitempty"`
	Records   RecordsDefs `json:"recordsDefs,omitempty"`
	GenOpts   GenOptsDef  `json:"gen_options,omitempty"`
}

type ValidationResult interface {
	Valid() bool
	ValidationErrors() []ValidationError
}

type ValidationError interface {
	String() string
}

type jsonValidationResult struct {
	result *gojsonschema.Result
}

func (v *jsonValidationResult) Valid() bool {
	return v.result.Valid()
}

func (v *jsonValidationResult) ValidationErrors() []ValidationError {
	jsonErrors := v.result.Errors()
	resultErrors := make([]ValidationError, len(v.result.Errors()))
	for index := range jsonErrors {
		resultErrors[index] = jsonErrors[index]
	}
	return resultErrors
}

type DefaultValidationResult struct {
	verrors []DefaultValidationError
}

type DefaultValidationError struct {
	fieldName       string
	validationError string
}

func (v *DefaultValidationResult) AddError(fieldName, errormsg string) *DefaultValidationResult {
	verr := DefaultValidationError{
		fieldName:       fieldName,
		validationError: errormsg,
	}
	v.verrors = append(v.verrors, verr)
	return v
}

func (v *DefaultValidationError) String() string {
	return fmt.Sprintf("field[%v], error[%v]", v.fieldName, v.validationError)
}

func (v *DefaultValidationResult) Valid() bool {
	return len(v.verrors) == 0
}

func (v *DefaultValidationResult) ValidationErrors() []ValidationError {

	resultErrors := make([]ValidationError, len(v.verrors))
	for index := range v.verrors {
		resultErrors[index] = &v.verrors[index]
	}
	return resultErrors
}

// Validate JSON of WOService Definition against WO JSON Schema
func validateJSONSchemaContent(defJsonContent []byte) (ValidationResult, error) {

	//schemaFile, err := os.Open("resources/schemas/wop_service.schema_latest_draft.json")
	//defer schemaFile.Close()
	content, err := resources.ReadFile("resources/schemas/wop_service.schema_latest_draft.json")
	if err != nil {
		return nil, err
	}
	//content, err := io.ReadAll(schemaFile)
	schemaLoader := gojsonschema.NewBytesLoader(content)

	documentLoader := gojsonschema.NewBytesLoader(defJsonContent)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}

	return &jsonValidationResult{result}, nil

}

// Validate JSON of WOService Definition against WO JSON Schema
func ValidateJSONSchema(defReader io.Reader) (ValidationResult, error) {
	serviceSpecs, err := readAndRemoveJSONComments(defReader)
	if err != nil {
		return nil, err
	}
	return validateJSONSchemaContent(serviceSpecs)
}

func readAndRemoveJSONComments(defReader io.Reader) ([]byte, error) {
	serviceSpecs, err := io.ReadAll(defReader)
	if err != nil {
		return nil, err
	}
	serviceSpecs = removeCppStyleComments(removeCStyleComments(serviceSpecs))
	return serviceSpecs, nil
}

func BuildSrvDefFromJSON(srvDefJsonReader io.Reader) (*SrvDef, error) {
	srvDef := &SrvDef{}
	serviceSpecs, err := readAndRemoveJSONComments(srvDefJsonReader)
	if err != nil {
		return srvDef, err
	}
	result, err := validateJSONSchemaContent(serviceSpecs)
	if err != nil {
		return srvDef, err
	}
	if !result.Valid() {
		//TODO return errors details once Error
		return srvDef, fmt.Errorf("invalid WOP Service Definiton:,%v", result.ValidationErrors())
	}

	err = json.Unmarshal([]byte(serviceSpecs), &srvDef)
	if err != nil {
		return srvDef, err
	}
	result, err = validateSrvDef(srvDef)
	if !result.Valid() {

		return srvDef, fmt.Errorf("invalid WOP SrvDef:,%v", result.ValidationErrors())
	}

	return srvDef, nil
}

func validateSrvDef(srvDef *SrvDef) (ValidationResult, error) {
	//TODO Refactor to common result error
	result := &DefaultValidationResult{}
	validateRecords(srvDef, result)
	validateCommands(srvDef, result)
	return result, nil
}

func validateRecords(srvDef *SrvDef, result *DefaultValidationResult) {
	records := srvDef.Records
	if len(records) > 0 {
		for _, record := range records {
			//TODO check for cycles
			validateRecord(srvDef, &record, result)
		}
	}
}

func validateRecord(srvDef *SrvDef, recDef *RecordDef, result *DefaultValidationResult) {
	validateFieldDefs(srvDef, recDef.Fields, recDef, result)
}

func validateCommands(srvDef *SrvDef, result *DefaultValidationResult) {
	commands := srvDef.Commands
	if len(commands) > 0 {
		for _, cmd := range commands {
			validateCommand(srvDef, &cmd, result)
		}
	}
}

func validateCommand(srvDef *SrvDef, cmdDef *CommandDef, result *DefaultValidationResult) {
	validateFieldDefs(srvDef, cmdDef.Fields, cmdDef, result)
}

func validateFieldDefs(srvDef *SrvDef, fields FieldDefs, fieldsOwner any, result *DefaultValidationResult) {

	if len(fields) > 0 {
		for _, field := range fields {
			validateFieldDef(srvDef, &field, fieldsOwner, result)
		}
	}
}

func validateFieldDef(srvDef *SrvDef, fieldDef *FieldDef, fieldsOwner any, result *DefaultValidationResult) {
	if fieldDef.Type == RECREF {
		ref := fieldDef.RecRef
		if !strings.HasPrefix(ref, "#/recordsDefs/") {
			result.AddError(fieldDef.Id, "fields with type 'recRef' must start with '#/recordsDefs/' ")
		}
		recordDefName := strings.TrimPrefix(ref, "#/recordsDefs/")
		if _, ok := srvDef.Records[recordDefName]; !ok {
			result.AddError(fieldDef.Id, fmt.Sprintf("could not find recordDef with name [%v] ", recordDefName))
		}

	}
}

func removeCStyleComments(content []byte) []byte {
	// http://blog.ostermiller.org/find-comment
	ccmt := regexp.MustCompile(`/\*([^*]|[\r\n]|(\*+([^*/]|[\r\n])))*\*+/`)
	return ccmt.ReplaceAll(content, []byte(""))
}

func removeCppStyleComments(content []byte) []byte {
	// Ugly but works, as regex solution not solid, e.g. breaks for http://

	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		l := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			buf.Write([]byte(l))
		} else {
			buf.WriteByte('\n')
		}
	}

	return buf.Bytes()
}

func MarshalSrvDefToJSON(w io.Writer, srvDef *SrvDef) error {
	content, err := json.Marshal(srvDef)
	if err != nil {
		return err
	}
	result, err := validateJSONSchemaContent(content)
	if err != nil {
		return err
	}
	if !result.Valid() {
		fmt.Print(string(content))
		return fmt.Errorf("invalid Srv Def, %v", result.ValidationErrors())
	}

	_, err = w.Write(content)
	if err != nil {
		return fmt.Errorf("error marshahlling SrvDef %v", err)
	}
	return nil
}

// The higher-order-function takes an array and a function as arguments
func mapForEachKeySorted[V any](m map[string]V, fn func(key string, value V) any) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fn(key, m[key])
	}
}

func sortBy[V any](arr []V, compareFunc func(iv, jv V) bool) []V {
	sorted := arr[:]

	sort.Slice(sorted, func(i, j int) bool {
		return compareFunc(sorted[i], sorted[j])
	})
	return sorted
}

func LoadTemplate(tmpltName string, tmpltEmbedPath string) (*template.Template, error) {

	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
	}

	//Open template

	tmpltSrc, err := resources.ReadFile(tmpltEmbedPath)
	if err != nil {
		return nil, err
	}

	if len(tmpltSrc) == 0 {
		return nil, errors.New("code gen template not found or empty")
	}
	tmpl, err := template.New(tmpltName).Funcs(funcMap).Parse(string(tmpltSrc))
	if err != nil {
		return nil, err
	}
	return tmpl, nil

}
