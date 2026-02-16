package mvgen

import (
	"strings"
)

// GenerateJSVanillaClasses generates plain JavaScript classes with JSDoc comments (no dependencies)
func GenerateJSVanillaClasses(srvDef *SrvDef) ([]byte, error) {
	return GenerateFromEmbeddTemplate(srvDef, "js_classes", "resources/codegen_templates/js/js_classes_code.txt")
}

// GenerateTSDefinitions generates TypeScript type definitions (no dependencies)
func GenerateTSDefinitions(srvDef *SrvDef) ([]byte, error) {
	return GenerateFromEmbeddTemplate(srvDef, "js_types", "resources/codegen_templates/js/js_types_code.txt")
}

// GenerateJSPackage generates the JavaScript package utilities
func GenerateJSPackage(srvDef *SrvDef) ([]byte, error) {
	return GenerateFromEmbeddTemplate(srvDef, "js_package", "resources/codegen_templates/js/js_package_code.txt")
}

// jsTypeBase returns the base JavaScript type for a field
func jsTypeBase(field FieldDef) string {
	switch field.Type {
	case "string":
		return "string"
	case "boolean":
		return "boolean"
	case "int32", "int64", "uint32", "sint32", "float", "double":
		return "number"
	case "bytes":
		return "Uint8Array"
	case "timestamp":
		return "Date"
	case "duration":
		return "number"
	case "uuid":
		return "string"
	case "map":
		valueType := field.MapValueType
		if valueType == "" {
			valueType = "string"
		}
		// Map JS type for value
		jsValueType := jsMapValueType(valueType)
		return "Object.<string, " + jsValueType + ">"
	case "recRef":
		recName := strings.TrimPrefix(field.RecRef, "#/recordsDefs/")
		return recName
	default:
		return "any"
	}
}

// jsMapValueType converts a map value type to JavaScript type
func jsMapValueType(valueType string) string {
	switch valueType {
	case "string":
		return "string"
	case "boolean":
		return "boolean"
	case "int32", "int64", "uint32", "sint32", "float", "double":
		return "number"
	default:
		return "any"
	}
}

// jsVerifyCheck returns the JavaScript verification check for a field type
func jsVerifyCheck(field FieldDef) string {
	switch field.Type {
	case "string", "uuid":
		return "typeof message.%s !== 'string'"
	case "boolean":
		return "typeof message.%s !== 'boolean'"
	case "int32", "int64", "uint32", "sint32", "float", "double", "duration":
		return "typeof message.%s !== 'number'"
	case "bytes":
		return "!(message.%s instanceof Uint8Array)"
	case "timestamp":
		return "!(message.%s instanceof Date)"
	case "map":
		return "typeof message.%s !== 'object'"
	case "recRef":
		return "typeof message.%s !== 'object'"
	default:
		return "false" // Skip unknown types
	}
}

// jsConvertValue returns the JavaScript conversion expression for a field type
func jsConvertValue(field FieldDef) string {
	switch field.Type {
	case "string", "uuid":
		return "String(obj.%s)"
	case "boolean":
		return "Boolean(obj.%s)"
	case "int32", "int64", "uint32", "sint32", "float", "double", "duration":
		return "Number(obj.%s)"
	case "bytes":
		return "obj.%s instanceof Uint8Array ? obj.%s : new Uint8Array(obj.%s)"
	case "timestamp":
		return "obj.%s instanceof Date ? obj.%s : new Date(obj.%s)"
	default:
		return "obj.%s"
	}
}

// jsDocTypeFromFieldDef returns the JSDoc type annotation for a field
func jsDocTypeFromFieldDef(field FieldDef) string {
	baseType := jsTypeBase(field)
	if field.Repeated {
		return "Array.<" + baseType + ">"
	}
	return baseType
}

// jsDefaultValue returns a default value expression for JavaScript
func jsDefaultValue(field FieldDef) string {
	if field.Repeated {
		return " ?? []"
	}
	switch field.Type {
	case "string", "uuid":
		return " ?? ''"
	case "boolean":
		return " ?? false"
	case "int32", "int64", "uint32", "sint32", "float", "double", "duration":
		return " ?? 0"
	case "bytes":
		return " ?? new Uint8Array()"
	case "timestamp":
		return " ?? null"
	case "map":
		return " ?? {}"
	case "recRef":
		return " ?? null"
	default:
		return " ?? null"
	}
}

// tsTypeBase returns the base TypeScript type for a field
func tsTypeBase(field FieldDef) string {
	switch field.Type {
	case "string":
		return "string"
	case "boolean":
		return "boolean"
	case "int32", "int64", "uint32", "sint32", "float", "double":
		return "number"
	case "bytes":
		return "Uint8Array"
	case "timestamp":
		return "Date"
	case "duration":
		return "number"
	case "uuid":
		return "string"
	case "map":
		valueType := field.MapValueType
		if valueType == "" {
			valueType = "string"
		}
		// Map TS type for value
		tsValueType := tsMapValueType(valueType)
		return "Record<string, " + tsValueType + ">"
	case "recRef":
		recName := strings.TrimPrefix(field.RecRef, "#/recordsDefs/")
		return recName
	default:
		return "unknown"
	}
}

// tsMapValueType converts a map value type to TypeScript type
func tsMapValueType(valueType string) string {
	switch valueType {
	case "string":
		return "string"
	case "boolean":
		return "boolean"
	case "int32", "int64", "uint32", "sint32", "float", "double":
		return "number"
	default:
		return "unknown"
	}
}

// tsTypeFromFieldDef returns the TypeScript type for a field
func tsTypeFromFieldDef(field FieldDef) string {
	baseType := tsTypeBase(field)
	if field.Repeated {
		return baseType + "[]"
	}
	// Add null union for types that default to null
	if field.Type == "timestamp" || field.Type == "recRef" || field.Type == "bytes" {
		return baseType + " | null"
	}
	return baseType
}

// tsTypeNullable returns the TypeScript type with | null for interface properties (protobufjs style)
func tsTypeNullable(field FieldDef) string {
	baseType := tsTypeBase(field)
	if field.Repeated {
		return baseType + "[] | null"
	}
	return baseType + " | null"
}

// tsDefaultValue returns a default value expression for TypeScript
func tsDefaultValue(field FieldDef) string {
	if field.Repeated {
		return " ?? []"
	}
	switch field.Type {
	case "string", "uuid":
		return " ?? ''"
	case "boolean":
		return " ?? false"
	case "int32", "int64", "uint32", "sint32", "float", "double", "duration":
		return " ?? 0"
	case "bytes":
		return " ?? new Uint8Array()"
	case "timestamp":
		return " ?? null"
	case "map":
		return " ?? {}"
	case "recRef":
		return " ?? null"
	default:
		return " ?? null"
	}
}

// isRequiredField determines if a field should be marked as required
// For now, all fields are optional to allow partial construction
func isRequiredField(field FieldDef) bool {
	return false
}

// isLastField checks if this is the last field in the list
func isLastField(index int, total int) bool {
	return index == total-1
}
