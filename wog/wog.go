package wog

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
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/workoak/wop/wou"
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
	INT64   FieldDataType = "int64"
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

type CommandDefs wou.OMap[string, CommandDef]
type FieldDefs wou.OMap[string, FieldDef]
type RecordsDefs wou.OMap[string, RecordDef]
type GenOptsDef wou.OMap[string, string]

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
	schemaFile := "resources/wopapi/wopspec/0.1/schema/2023-09-19.json"
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
	err = c.AddResource("https://spec.workoak.com/wopspec/0.1/schema/2023-09-19", schemaJson)
	if err != nil {
		return nil, err
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(defJsonContent))
	if err != nil {
		return nil, err
	}
	jsonMap, ok := inst.(map[string]interface{})

	if !ok {
		return nil, errors.New("invalid JSON")
	}
	jsonSchema := schemaFile
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
