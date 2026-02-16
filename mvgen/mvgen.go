package mvgen

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/mainvec/ugo/omap"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed resources
var resources embed.FS

type GenOpt struct {
	OptName  string
	OptValue string
}

//type CommandId string

type CommandDef struct {
	Id           string    `json:"-"`
	Title        string    `json:"title,omitempty"`
	Alias        string    `json:"alias,omitempty"`
	Desc         string    `json:"desc,omitempty"`
	Fields       FieldDefs `json:"fields,omitempty"`
	ResultFields FieldDefs `json:"resultFields,omitempty"`
}

type RecordDef struct {
	Name   string    `json:"name"`
	Title  string    `json:"title,omitempty"`
	Desc   string    `json:"desc,omitempty"`
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
	INT64   FieldDataType = "int64"
	RECREF  FieldDataType = "recRef"
	MAP     FieldDataType = "map"
)

type FieldDef struct {
	Id           string        `json:"-"`
	Title        string        `json:"title,omitempty"`
	Alias        string        `json:"alias,omitempty"`
	Desc         string        `json:"desc,omitempty"`
	Fnum         int32         `json:"fnum"`
	Type         FieldDataType `json:"type"`
	Repeated     bool          `json:"repeated,omitempty"`
	RecRef       string        `json:"$ref,omitempty"`
	MapValueType string        `json:"valueType,omitempty"`
}

type CommandDefs omap.OMap[string, CommandDef]
type FieldDefs omap.OMap[string, FieldDef]
type RecordsDefs omap.OMap[string, RecordDef]
type GenOptsDef omap.OMap[string, string]

type SrvDef struct {
	Id         string      `json:"$id"`
	Name       string      `json:"name"`
	Namespace  string      `json:"namespace"`
	Title      string      `json:"title,omitempty"`
	Base       string      `json:"base,omitempty"`
	Desc       string      `json:"desc,omitempty"`
	Version    string      `json:"version,omitempty"`
	Commands   CommandDefs `json:"commands,omitempty"`
	Records    RecordsDefs `json:"recordsDefs,omitempty"`
	GenOpts    GenOptsDef  `json:"gen_options,omitempty"`
	ProtocOpts []string    `json:"-"` //Transient Holder for now, filled why processing options
}

type ValidationResult interface {
	Valid() bool
	ValidationErrors() []ValidationError
}

type ValidationError interface {
	String() string
}
type JSONValidationError struct {
	vErr *jsonschema.ValidationError
}

func (v *JSONValidationError) String() string {
	p := v.vErr.ErrorKind.KeywordPath()
	l := v.vErr.InstanceLocation
	return fmt.Sprintf("error[%v],path[%v],loc[%v]", v.vErr.Error(), p, l)

}

type jsonValidationResult struct {
	valid              bool
	schValidationError *jsonschema.ValidationError
}

func (v *jsonValidationResult) Valid() bool {
	return v.valid
}

func (v *jsonValidationResult) ValidationErrors() []ValidationError {
	if v.schValidationError == nil {
		return nil
	}
	jsonErrors := v.schValidationError.Causes
	resultErrors := make([]ValidationError, len(jsonErrors))
	for index := range jsonErrors {
		resultErrors[index] = &JSONValidationError{vErr: jsonErrors[index]}
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

type HTTPURLLoader http.Client

func (l *HTTPURLLoader) Load(url string) (any, error) {
	client := (*http.Client)(l)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%s returned status code %d", url, resp.StatusCode)
	}
	defer resp.Body.Close()

	return jsonschema.UnmarshalJSON(resp.Body)
}

func newHTTPURLLoader(insecure bool) *HTTPURLLoader {
	httpLoader := HTTPURLLoader(http.Client{
		Timeout: 15 * time.Second,
	})
	if insecure {
		httpLoader.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &httpLoader
}

// Validate JSON of WOService Definition against WO JSON Schema
func validateJSONSchemaContent(defJsonContent []byte) (ValidationResult, error) {
	loader := jsonschema.SchemeURLLoader{
		"file":  jsonschema.FileLoader{},
		"http":  newHTTPURLLoader(false),
		"https": newHTTPURLLoader(false),
	}

	c := jsonschema.NewCompiler()
	c.UseLoader(loader)

	vr, err := addResource("https://spec.mainvec.com/mvepspec/0.1/schema/2023-09-19", "resources/mvepspec/0.1/schema/2023-09-19.json", c)
	if err != nil {
		return vr, err
	}
	vr, err = addResource("https://spec.mainvec.com/mvepspec/0.1/schema/2026-01-15", "resources/mvepspec/0.1/schema/2026-01-15.json", c)
	if err != nil {
		return vr, err
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(defJsonContent))
	if err != nil {
		return nil, err
	}
	jsonMap, ok := inst.(map[string]interface{})

	if !ok {
		return nil, errors.New("invalid JSON")
	}
	jsonSchema := ""
	if schm, ok := jsonMap["$schema"]; ok {
		jsonSchema, ok = schm.(string)
		if !ok {
			return nil, errors.New("invalid JSON")

		}
	}

	sch, err := c.Compile(jsonSchema)

	if err != nil {

		return nil, err
	}

	err = sch.Validate(inst)
	if err != nil {
		vErr, ok := err.(*jsonschema.ValidationError)
		if !ok {
			return nil, err
		}
		return &jsonValidationResult{valid: false, schValidationError: vErr}, nil
	}

	return &jsonValidationResult{valid: true}, nil

}

func addResource(resourceUrl string, schemaFile string, c *jsonschema.Compiler) (ValidationResult, error) {
	schema, err := resources.Open(schemaFile)
	if err != nil {
		return nil, err
	}
	defer schema.Close()

	schemaBytes, err := resources.ReadFile(schemaFile)
	if err != nil {
		return nil, err
	}
	schemaJson, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return nil, err
	}
	err = c.AddResource(schemaFile, schemaJson)
	if err != nil {
		return nil, err
	}
	err = c.AddResource(resourceUrl, schemaJson)
	if err != nil {
		return nil, err
	}
	return nil, nil
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
		return srvDef, fmt.Errorf("invalid MVEP Service Definiton:,%v", result.ValidationErrors())
	}

	err = json.Unmarshal([]byte(serviceSpecs), &srvDef)
	if err != nil {
		return srvDef, err
	}
	result, err = validateSrvDef(srvDef)
	if !result.Valid() {

		return srvDef, fmt.Errorf("invalid MVEP SrvDef:,%v", result.ValidationErrors())
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
	validateFieldDefs(srvDef, cmdDef.ResultFields, cmdDef, result)
}

func validateFieldDefs(srvDef *SrvDef, fields FieldDefs, fieldsOwner any, result *DefaultValidationResult) {

	if len(fields) > 0 {
		for _, field := range fields {
			validateFieldDef(srvDef, &field, fieldsOwner, result)
		}
	}
}

func validateFieldDef(srvDef *SrvDef, fieldDef *FieldDef, fieldsOwner any, result *DefaultValidationResult) {
	_ = fieldsOwner
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

func LoadTemplate(tmpltName string, templateReader io.Reader) (*template.Template, error) {

	funcMap := template.FuncMap{
		"ToUpper": strings.ToUpper,
		"ToTitle": strings.ToTitle,
		"ToCamel": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			r := []rune(s)
			r[0] = unicode.ToUpper(r[0])
			return string(r)
		},
		"PrintStdOut": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			fmt.Fprintln(os.Stdout, s)
			return ""
		},
		"PrintStdErr": func(s string) string {
			if len(s) == 0 {
				return ""
			}
			fmt.Fprintln(os.Stderr, s)
			return ""
		},
		// GoType maps FieldDataType to Go type string for vanilla struct generation
		"GoType": func(field FieldDef) string {
			return goTypeFromFieldDef(field)
		},
		// GoTypeBase returns the base Go type without slice wrapper
		"GoTypeBase": func(field FieldDef) string {
			return goTypeBase(field)
		},
		// RecRefName extracts the record name from a $ref like "#/recordsDefs/Address"
		"RecRefName": func(ref string) string {
			return strings.TrimPrefix(ref, "#/recordsDefs/")
		},
		// NeedsTimeImport checks if any field requires the time package
		"NeedsTimeImport": NeedsTimeImport,
		// NeedsUUIDImport checks if any field requires the uuid package
		"NeedsUUIDImport": NeedsUUIDImport,
		// SortFieldsByFnum returns fields sorted by their fnum value
		"SortFieldsByFnum": SortFieldsByFnum,
		// GoZeroValue returns the zero value for a field type
		"GoZeroValue": goZeroValue,
		// JavaScript/TypeScript template functions
		// JSDocType returns the JSDoc type annotation for a field
		"JSDocType": func(field FieldDef) string {
			return jsDocTypeFromFieldDef(field)
		},
		// JSDefaultValue returns a default value expression for JavaScript
		"JSDefaultValue": jsDefaultValue,
		// JSVerifyCheck returns the JavaScript verification check for a field
		"JSVerifyCheck": func(field FieldDef) string {
			return jsVerifyCheck(field)
		},
		// JSConvertValue returns the JavaScript conversion expression for a field
		"JSConvertValue": func(field FieldDef) string {
			return jsConvertValue(field)
		},
		// TSType returns the TypeScript type for a field
		"TSType": func(field FieldDef) string {
			return tsTypeFromFieldDef(field)
		},
		// TSTypeNullable returns the TypeScript type with | null (protobufjs style)
		"TSTypeNullable": func(field FieldDef) string {
			return tsTypeNullable(field)
		},
		// TSDefaultValue returns a default value expression for TypeScript
		"TSDefaultValue": tsDefaultValue,
		// IsRequiredField determines if a field should be marked as required
		"IsRequiredField": isRequiredField,
		// IsLastField checks if this is the last field in the list
		"IsLastField": isLastField,
	}

	//Open template
	tmpltSrc, err := io.ReadAll(templateReader)
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

// goTypeBase returns the base Go type for a field (without slice wrapper)
func goTypeBase(field FieldDef) string {
	switch field.Type {
	case "string":
		return "string"
	case "boolean":
		return "bool"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "bytes":
		return "[]byte"
	case "uint32":
		return "uint32"
	case "sint32":
		return "int32"
	case "timestamp":
		return "time.Time"
	case "duration":
		return "time.Duration"
	case "uuid":
		return "uuid.UUID"
	case "map":
		valueType := field.MapValueType
		if valueType == "" {
			valueType = "string"
		}
		return "map[string]" + valueType
	case "recRef":
		recName := strings.TrimPrefix(field.RecRef, "#/recordsDefs/")
		return "*" + recName
	default:
		return "any"
	}
}

// goTypeFromFieldDef returns the full Go type including slice wrapper for repeated fields
func goTypeFromFieldDef(field FieldDef) string {
	baseType := goTypeBase(field)
	if field.Repeated {
		return "[]" + baseType
	}
	return baseType
}

// NeedsTimeImport checks if any field in the SrvDef requires the time package
func NeedsTimeImport(srvDef *SrvDef) bool {
	// Check commands
	for _, cmd := range srvDef.Commands {
		for _, field := range cmd.Fields {
			if field.Type == "timestamp" || field.Type == "duration" {
				return true
			}
		}
		for _, field := range cmd.ResultFields {
			if field.Type == "timestamp" || field.Type == "duration" {
				return true
			}
		}
	}
	// Check records
	for _, rec := range srvDef.Records {
		for _, field := range rec.Fields {
			if field.Type == "timestamp" || field.Type == "duration" {
				return true
			}
		}
	}
	return false
}

// NeedsUUIDImport checks if any field in the SrvDef requires the uuid package
func NeedsUUIDImport(srvDef *SrvDef) bool {
	// Check commands
	for _, cmd := range srvDef.Commands {
		for _, field := range cmd.Fields {
			if field.Type == "uuid" {
				return true
			}
		}
		for _, field := range cmd.ResultFields {
			if field.Type == "uuid" {
				return true
			}
		}
	}
	// Check records
	for _, rec := range srvDef.Records {
		for _, field := range rec.Fields {
			if field.Type == "uuid" {
				return true
			}
		}
	}
	return false
}

// NamedField represents a field with its name for sorting purposes
type NamedField struct {
	Name  string
	Field FieldDef
}

// SortFieldsByFnum returns fields sorted by their fnum value
func SortFieldsByFnum(fields FieldDefs) []NamedField {
	result := make([]NamedField, 0, len(fields))
	for name, field := range fields {
		result = append(result, NamedField{Name: name, Field: field})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Field.Fnum < result[j].Field.Fnum
	})
	return result
}

// goZeroValue returns the zero value for a field type as a string
func goZeroValue(field FieldDef) string {
	// Handle repeated fields (slices)
	if field.Repeated {
		return "nil"
	}

	switch field.Type {
	case "string":
		return `""`
	case "boolean":
		return "false"
	case "int32", "int64", "uint32", "sint32":
		return "0"
	case "float":
		return "0.0"
	case "double":
		return "0.0"
	case "bytes":
		return "nil"
	case "timestamp":
		return "time.Time{}"
	case "duration":
		return "0"
	case "uuid":
		return "uuid.UUID{}"
	case "map":
		return "nil"
	case "recRef":
		return "nil"
	default:
		//lets panic so this dets added
		panic("unknown go zero value for:" + field.Type)
	}
}
