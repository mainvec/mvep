package toolkit

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
	Group        string    `json:"group,omitempty"` // command group path, /-separated
	Desc         string    `json:"desc,omitempty"`
	Fields       FieldDefs `json:"fields,omitempty"`
	ResultFields FieldDefs `json:"resultFields,omitempty"`
}

// GroupDef carries optional metadata for a command group, keyed by full
// group path in SrvDef.CommandGroups. All fields are optional; a
// group referenced by a command but absent from CommandGroups is auto-created
// with the path segment as its name and no metadata.
type GroupDef struct {
	Title   string   `json:"title,omitempty"`
	Desc    string   `json:"desc,omitempty"`
	Aliases []string `json:"aliases,omitempty"`
	Hidden  bool     `json:"hidden,omitempty"`
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
	Tags         []string      `json:"tags,omitempty"`
}

type CommandDefs omap.OMap[string, CommandDef]
type FieldDefs omap.OMap[string, FieldDef]
type RecordsDefs omap.OMap[string, RecordDef]
type GenOptsDef omap.OMap[string, string]
type GroupDefs omap.OMap[string, GroupDef]

// Get returns the CommandDef for key k and ok=true if present.
func (c CommandDefs) Get(k string) (CommandDef, bool) {
	v, ok := c[k]
	return v, ok
}

// Get returns the GroupDef for key k and ok=true if present.
func (g GroupDefs) Get(k string) (GroupDef, bool) {
	v, ok := g[k]
	return v, ok
}

// Get returns the FieldDef for key k and ok=true if present.
func (f FieldDefs) Get(k string) (FieldDef, bool) {
	v, ok := f[k]
	return v, ok
}

// Get returns the RecordDef for key k and ok=true if present.
func (r RecordsDefs) Get(k string) (RecordDef, bool) {
	v, ok := r[k]
	return v, ok
}

// Get returns the string value for key k and ok=true if present.
func (g GenOptsDef) Get(k string) (string, bool) {
	v, ok := g[k]
	return v, ok
}

type SrvDef struct {
	Id            string      `json:"$id"`
	Name          string      `json:"name"`
	Namespace     string      `json:"namespace"`
	Title         string      `json:"title,omitempty"`
	Base          string      `json:"base,omitempty"`
	Desc          string      `json:"desc,omitempty"`
	Version       string      `json:"version,omitempty"`
	Commands      CommandDefs `json:"commands,omitempty"`
	CommandGroups GroupDefs   `json:"commandGroups,omitempty"`
	Records       RecordsDefs `json:"recordsDefs,omitempty"`
	GenOpts       GenOptsDef  `json:"gen_options,omitempty"`
	ProtocOpts    []string    `json:"-"` //Transient Holder for now, filled why processing options
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

type schemaResource struct {
	URL          string
	ResourceFile string
}

var supportedSchemaResources = []schemaResource{
	{
		URL:          "https://spec.mainvec.com/mvepspec/0.1/schema/2023-09-19",
		ResourceFile: "resources/mvepspec/0.1/schema/2023-09-19.json",
	},
	{
		URL:          "https://spec.mainvec.com/mvepspec/0.1/schema/2026-01-15",
		ResourceFile: "resources/mvepspec/0.1/schema/2026-01-15.json",
	},
	{
		URL:          "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15",
		ResourceFile: "resources/mvepspec/0.2/schema/2026-01-15.json",
	},
	// Alias: legacy mvpspec/0.2 URL kept resolvable so existing spec files that
	// pin the old $schema continue to validate. Prefer mvepspec/0.2 for new specs.
	{
		URL:          "https://spec.mainvec.com/mvpspec/0.2/schema/2026-01-15",
		ResourceFile: "resources/mvpspec/0.2/schema/2026-01-15.json",
	},
}

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

	for _, schemaResource := range supportedSchemaResources {
		err := addResource(schemaResource.URL, schemaResource.ResourceFile, c)
		if err != nil {
			return nil, err
		}
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(defJsonContent))
	if err != nil {
		return nil, err
	}
	jsonMap, ok := inst.(map[string]interface{})

	if !ok {
		return nil, errors.New("invalid JSON")
	}
	jsonSchema := "https://spec.mainvec.com/mvepspec/0.2/schema/2026-01-15"
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

func addResource(resourceURL string, schemaFile string, c *jsonschema.Compiler) error {
	schema, err := resources.Open(schemaFile)
	if err != nil {
		return err
	}
	defer schema.Close()

	schemaBytes, err := resources.ReadFile(schemaFile)
	if err != nil {
		return err
	}
	schemaJson, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		return err
	}
	err = c.AddResource(schemaFile, schemaJson)
	if err != nil {
		return err
	}
	err = c.AddResource(resourceURL, schemaJson)
	if err != nil {
		return err
	}
	return nil
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
		// Descriptor emission helpers (plan 025, T3)
		"SortedCommandNames": sortedCommandNames,
		"SortedRecordNames":  sortedRecordNames,
		"GoFieldTypeEnum":    goFieldTypeEnum,
		"FieldIsRequired":    fieldIsRequired,
		"GoStringLit":        goStringLit,
		"GoStringSliceLit":   goStringSliceLit,
		"CmdDescOrTitle":     cmdDescOrTitle,
		// Command-group descriptor emission (plan 040, T4)
		"GroupDescs":       GroupDescs,
		"SortedGroupNames": sortedGroupNames,
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

// --- Descriptor emission helpers (plan 025, T3) ---

// sortedCommandNames returns command names in deterministic (sorted-key) order.
// The descriptor is emitted through this rather than ranging the omap directly
// so help/iteration order is defined (omap.OMap is a plain map).
func sortedCommandNames(cmds CommandDefs) []string {
	names := make([]string, 0, len(cmds))
	it := omap.IteratorByKey(cmds)
	for it.HasNext() {
		k, _ := it.Next()
		names = append(names, k)
	}
	return names
}

// sortedRecordNames returns record names in deterministic (sorted-key) order.
func sortedRecordNames(recs RecordsDefs) []string {
	names := make([]string, 0, len(recs))
	it := omap.IteratorByKey(recs)
	for it.HasNext() {
		k, _ := it.Next()
		names = append(names, k)
	}
	return names
}

// sortedGroupNames returns command-group paths in deterministic (sorted-key)
// order, so the descriptor's Groups slice is emitted deterministically. Group
// ordering matters: cli.New builds parents in Groups order, so help output and
// the parent-of finding must be stable (plan 040 T4).
func sortedGroupNames(groups GroupDefs) []string {
	names := make([]string, 0, len(groups))
	it := omap.IteratorByKey(groups)
	for it.HasNext() {
		k, _ := it.Next()
		names = append(names, k)
	}
	return names
}

// GroupDesc is the toolkit-side projection of a group for descriptor emission.
// The runtime mvep.GroupDesc is authored by codegen from this.
type GroupDesc struct {
	Path    string
	Name    string
	Title   string
	Desc    string
	Aliases []string
	Hidden  bool
}

// orderedGroupDescs computes the full, deterministic, flat group list for
// descriptor emission: every declared commandGroups entry plus every group
// referenced by a command, with intermediate segments auto-created so the
// descriptor is complete on its own (plan 040 T4). Paths are sorted, which
// guarantees a parent sorts before its children. Metadata for an undeclared
// group defaults to the last path segment as Name and empty metadata.
func orderedGroupDescs(srvDef *SrvDef) []GroupDesc {
	seen := map[string]bool{}
	var out []GroupDesc
	add := func(path string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		gd := GroupDesc{Path: path}
		if len(path) > 0 {
			segs := strings.Split(path, "/")
			gd.Name = segs[len(segs)-1]
		}
		if meta, ok := srvDef.CommandGroups[path]; ok {
			gd.Title = meta.Title
			gd.Desc = meta.Desc
			gd.Aliases = meta.Aliases
			gd.Hidden = meta.Hidden
		}
		out = append(out, gd)
	}
	// Add each path's full segment chain (parents before children).
	chain := func(path string) {
		if path == "" {
			return
		}
		prefix := ""
		for _, seg := range strings.Split(path, "/") {
			if prefix == "" {
				prefix = seg
			} else {
				prefix = prefix + "/" + seg
			}
			add(prefix)
		}
	}
	for _, p := range sortedGroupNames(srvDef.CommandGroups) {
		chain(p)
	}
	it := omap.IteratorByKey(srvDef.Commands)
	for it.HasNext() {
		_, cmd := it.Next()
		chain(cmd.Group)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// GroupDescs exposes orderedGroupDescs to templates (plan 040 T4).
func GroupDescs(srvDef *SrvDef) []GroupDesc {
	return orderedGroupDescs(srvDef)
}

// goFieldTypeEnum maps a spec FieldDataType to its runtime mvep.FieldType
// constant name. Unknown types panic: an unrepresentable construct must fail
// generation loudly, never emit a wrong descriptor (T5 hardens this into a
// returned error).
func goFieldTypeEnum(field FieldDef) string {
	switch field.Type {
	case "string":
		return "FieldString"
	case "boolean":
		return "FieldBool"
	case "int32":
		return "FieldInt32"
	case "int64":
		return "FieldInt64"
	case "uint32":
		return "FieldUint32"
	case "sint32":
		return "FieldSint32"
	case "float":
		return "FieldFloat"
	case "double":
		return "FieldDouble"
	case "bytes":
		return "FieldBytes"
	case "timestamp":
		return "FieldTimestamp"
	case "duration":
		return "FieldDuration"
	case "uuid":
		return "FieldUUID"
	case "map":
		return "FieldMap"
	case "recRef":
		return "FieldRecord"
	default:
		panic("goFieldTypeEnum: unsupported field type: " + string(field.Type))
	}
}

// descriptorSupportedFieldTypes is the set of spec field types the runtime
// descriptor can represent. A spec using a type outside this set fails
// generation with a clear error (validateDescriptorRepresentable) rather
// than panicking deep in template execution. Kept in sync with goFieldTypeEnum.
var descriptorSupportedFieldTypes = map[FieldDataType]bool{
	"string":    true,
	"boolean":   true,
	"int32":     true,
	"int64":     true,
	"uint32":    true,
	"sint32":    true,
	"float":     true,
	"double":    true,
	"bytes":     true,
	"timestamp": true,
	"duration":  true,
	"uuid":      true,
	"map":       true,
	"recRef":    true,
}

// validateDescriptorRepresentable walks every command and record field in the
// spec and rejects any construct the runtime descriptor cannot represent,
// returning an error that names the offending command (or record) and field.
// This is the T5 generate-time hard error: unrepresentable specs fail here,
// at mvep generate with the spec in hand, rather than panicking in a template
// or failing silently at runtime.
func validateDescriptorRepresentable(srvDef *SrvDef) error {
	// Commands: iterate by key so the command name is available for the error.
	it := omap.IteratorByKey(srvDef.Commands)
	for it.HasNext() {
		cmdName, cmd := it.Next()
		if err := checkFieldsRepresentable(cmd.Fields, "command", cmdName); err != nil {
			return err
		}
		if err := checkFieldsRepresentable(cmd.ResultFields, "command", cmdName); err != nil {
			return err
		}
	}
	// Records: iterate by key so the record name is available for the error.
	rit := omap.IteratorByKey(srvDef.Records)
	for rit.HasNext() {
		recName, rec := rit.Next()
		if err := checkFieldsRepresentable(rec.Fields, "record", recName); err != nil {
			return err
		}
	}
	// Command groups: reject collisions and malformed paths (plan 040 T5).
	if err := validateCommandGroups(srvDef); err != nil {
		return err
	}
	return nil
}

// validateCommandGroups enforces the plan 040 generate-time group rules. Every
// error names the spec path, the group, and the colliding command so the
// developer holding the spec can fix it without digging.
func validateCommandGroups(srvDef *SrvDef) error {
	// leafName is the name a command resolves to under its parent: the alias
	// when present, else the snake_case of the command name (mirrors
	// runtime cli.commandName).
	leafName := func(cmd CommandDef, cmdName string) string {
		if cmd.Alias != "" {
			return cmd.Alias
		}
		return toSnake(cmdName)
	}

	// nameByParent maps "parentPath/leafName" -> command name for duplicate
	// detection. parentPath is the group path ("" for root).
	nameByParent := map[string]string{}
	// referencedGroups records every group path a command references.
	referencedGroups := map[string]bool{}

	it := omap.IteratorByKey(srvDef.Commands)
	for it.HasNext() {
		cmdName, cmd := it.Next()
		group := cmd.Group

		// Malformed path: empty segment, or leading/trailing slash.
		if group != "" {
			if strings.HasPrefix(group, "/") || strings.HasSuffix(group, "/") {
				return fmt.Errorf("command %q: group path %q must not have a leading or trailing '/'", cmdName, group)
			}
			for _, seg := range strings.Split(group, "/") {
				if seg == "" {
					return fmt.Errorf("command %q: group path %q contains an empty segment", cmdName, group)
				}
			}
			// Seed referencedGroups with every prefix of the path, mirroring
			// orderedGroupDescs' chain, so a titled intermediate segment is not
			// reported unreferenced (#43). The exact group and all its parents
			// count as referenced by this command.
			prefix := ""
			for _, seg := range strings.Split(group, "/") {
				if prefix == "" {
					prefix = seg
				} else {
					prefix = prefix + "/" + seg
				}
				referencedGroups[prefix] = true
			}
		}

		leaf := leafName(cmd, cmdName)
		key := group + "/" + leaf
		if prev, dup := nameByParent[key]; dup {
			return fmt.Errorf(
				"commands %q and %q both resolve to name %q under group %q",
				prev, cmdName, leaf, group,
			)
		}
		nameByParent[key] = cmdName
	}

	// Build the set of group names and aliases at each parent, and check
	// collisions between a group and a sibling command's name/alias.
	// groupNamesByParent: parentPath -> set of group leaf names.
	// groupAliasesByParent: parentPath -> map[alias]groupPath.
	groupNamesByParent := map[string]map[string]bool{}
	groupAliasesByParent := map[string]map[string]string{}
	groupPathByLeaf := map[string]string{} // "parent/leaf" -> full group path

	// Register all group paths (declared + referenced), including intermediate
	// segments, so a group can collide with a sibling group. Build the prefix
	// chains into a second map rather than mutating allGroups while ranging
	// over it (#45).
	allGroups := map[string]bool{}
	for _, p := range sortedGroupNames(srvDef.CommandGroups) {
		allGroups[p] = true
	}
	for g := range referencedGroups {
		allGroups[g] = true
	}
	expanded := map[string]bool{}
	for g := range allGroups {
		prefix := ""
		for _, seg := range strings.Split(g, "/") {
			if prefix == "" {
				prefix = seg
			} else {
				prefix = prefix + "/" + seg
			}
			expanded[prefix] = true
		}
	}
	for p := range expanded {
		allGroups[p] = true
	}

	for g := range allGroups {
		parent, leaf := splitGroupPath(g)
		if groupNamesByParent[parent] == nil {
			groupNamesByParent[parent] = map[string]bool{}
		}
		groupNamesByParent[parent][leaf] = true
		groupPathByLeaf[parent+"/"+leaf] = g
		// Group aliases come from declared metadata only.
		if meta, ok := srvDef.CommandGroups[g]; ok {
			if groupAliasesByParent[parent] == nil {
				groupAliasesByParent[parent] = map[string]string{}
			}
			for _, a := range meta.Aliases {
				groupAliasesByParent[parent][a] = g
			}
		}
	}

	// A group name or alias must not collide with a sibling command's name or
	// alias. A command's leaf name under its parent is its alias or snake_case.
	// A command's parent IS its group path (empty for root), not the path minus
	// its last segment (#42): a command under "server" is a sibling of the
	// "server/keys" group, not of the "server" group's siblings.
	cmdLeafByParent := map[string]map[string]string{} // parent -> leaf -> cmdName
	it2 := omap.IteratorByKey(srvDef.Commands)
	for it2.HasNext() {
		cmdName, cmd := it2.Next()
		parent := cmd.Group
		leaf := leafName(cmd, cmdName)
		if cmdLeafByParent[parent] == nil {
			cmdLeafByParent[parent] = map[string]string{}
		}
		cmdLeafByParent[parent][leaf] = cmdName
		// A command with an alias also registers the snake_case descriptor name
		// as a secondary ugo alias (cli.aliasesFor), so it can collide with a
		// group too (#45).
		if cmd.Alias != "" {
			secondary := toSnake(cmdName)
			if secondary != leaf {
				cmdLeafByParent[parent][secondary] = cmdName
			}
		}
	}

	for parent, leaves := range cmdLeafByParent {
		for leaf, cmdName := range leaves {
			if groupNamesByParent[parent][leaf] {
				return fmt.Errorf(
					"command %q name/alias %q collides with group %q under %q",
					cmdName, leaf, groupPathByLeaf[parent+"/"+leaf], parent,
				)
			}
			if gpath, ok := groupAliasesByParent[parent][leaf]; ok {
				return fmt.Errorf(
					"command %q name/alias %q collides with an alias of group %q under %q",
					cmdName, leaf, gpath, parent,
				)
			}
		}
	}

	// A declared commandGroups entry must be referenced by at least one
	// command; an unreferenced entry is almost always a typo.
	for _, p := range sortedGroupNames(srvDef.CommandGroups) {
		if !referencedGroups[p] {
			return fmt.Errorf("commandGroups entry %q is not referenced by any command", p)
		}
	}

	return nil
}

// splitGroupPath splits a group path into its parent path and final segment.
// The root's parent is "" and leaf is the whole path.
func splitGroupPath(path string) (parent, leaf string) {
	if path == "" {
		return "", ""
	}
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "", path
	}
	return path[:idx], path[idx+1:]
}

// toSnake converts CamelCase to snake_case (mirrors runtime cli.toSnake).
func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// checkFieldsRepresentable checks every field in a FieldDefs for descriptor
// representability, naming the owning command or record and the field.
func checkFieldsRepresentable(fields FieldDefs, ownerKind, ownerName string) error {
	fit := omap.IteratorByKey(fields)
	for fit.HasNext() {
		fieldName, field := fit.Next()
		if !descriptorSupportedFieldTypes[field.Type] {
			return fmt.Errorf(
				"descriptor cannot represent field %q (type %q) on %s %q: supported types are string, boolean, int32, int64, uint32, sint32, float, double, bytes, timestamp, duration, uuid, map, recRef",
				fieldName, field.Type, ownerKind, ownerName,
			)
		}
	}
	return nil
}

// fieldIsRequired reports whether a field is required. Required-ness is
// tag-derived in the current spec (`tags: ["required"]`).
func fieldIsRequired(field FieldDef) bool {
	for _, tag := range field.Tags {
		if tag == "required" {
			return true
		}
	}
	return false
}

// goStringLit returns s as a quoted Go string literal.
func goStringLit(s string) string {
	return fmt.Sprintf("%q", s)
}

// goStringSliceLit renders a []string as a Go composite literal, e.g.
// []string{"a", "b"}. Nil/empty renders as nil.
func goStringSliceLit(ss []string) string {
	if len(ss) == 0 {
		return "nil"
	}
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = goStringLit(s)
	}
	return "[]string{" + strings.Join(parts, ", ") + "}"
}

// cmdDescOrTitle returns desc if non-empty, otherwise title. The spec's
// commands carry a "title" but often no "desc"; the descriptor's Desc field
// is what the CLI shows as the command's Short description, so falling back
// to title ensures the help output is populated.
func cmdDescOrTitle(desc, title string) string {
	if desc != "" {
		return desc
	}
	return title
}
