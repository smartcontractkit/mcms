package canton

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/smartcontractkit/mcms/sdk"
	"github.com/smartcontractkit/mcms/types"
)

// Decoder decodes Canton MCMS operations into human-readable calls.
//
// Encoding (what we reverse): the deployment op calls MarshalHex() on the choice-argument struct,
// which returns raw wire bytes. Those bytes are hex-encoded and stored as operationData in
// AdditionalFields; the same raw bytes are stored in tx.Data via hex.DecodeString(operationData).
// Decoding: hex-encode tx.Data back to a hex string, call UnmarshalHex on the correct generated
// struct (found via the reflection registry), and read back its fields.
type Decoder struct{}

var _ sdk.Decoder = &Decoder{}

func NewDecoder() *Decoder {
	return &Decoder{}
}

func (d Decoder) Decode(tx types.Transaction, contractInterfaces string) (sdk.DecodedOperation, error) {
	var af AdditionalFields
	if err := json.Unmarshal(tx.AdditionalFields, &af); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Canton additional fields: %w", err)
	}
	if err := af.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Canton additional fields: %w", err)
	}
	if af.FunctionName == "" {
		return nil, fmt.Errorf("canton operation has no functionName to decode")
	}

	contractType := contractTypeFromFields(af, contractInterfaces, tx.ContractType)

	decoded, err := decodeOperationData(contractType, af.FunctionName, tx.Data)
	if err != nil {
		return nil, err
	}

	keys, args := fieldsOf(decoded)

	return NewDecodedOperation(contractType, af.FunctionName, keys, args)
}

// contractTypeFromFields resolves the Daml template entity name (e.g. "BurnMintTokenPool") used
// as the registry key. Sources in priority order:
//  1. AdditionalFields.TargetTemplateID ("#pkg:Module:Entity") — always present on real proposals.
//  2. contractInterfaces — the string passed by the caller (CLDF passes the resolved entity name here).
//  3. txContractType — tx.ContractType set by the deployment op (e.g. "CCIPFactory", "Executor").
func contractTypeFromFields(af AdditionalFields, contractInterfaces, txContractType string) string {
	if af.TargetTemplateID != "" {
		if _, _, entity, err := ParseTemplateIDFromString(af.TargetTemplateID); err == nil && entity != "" {
			return entity
		}
	}

	if contractInterfaces != "" {
		return contractInterfaces
	}

	return txContractType
}

// fieldsOf reflects over a decoded choice-argument struct and returns its field names
// (preferring the JSON tag) and display-friendly values in declaration order.
func fieldsOf(v any) ([]string, []any) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, nil
	}

	rt := rv.Type()
	keys := make([]string, 0, rt.NumField())
	args := make([]any, 0, rt.NumField())
	for i := range rt.NumField() {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Name
		if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
			name = strings.Split(tag, ",")[0]
		}
		keys = append(keys, name)
		args = append(args, toDisplayArg(rv.Field(i)))
	}

	return keys, args
}

// toDisplayArg converts a reflected Daml field value into a display-friendly Go value.
//
// Daml scalar types (TEXT, PARTY, CONTRACT_ID, NUMERIC, INT64, BOOL) are all type aliases for Go
// primitives, so we convert by kind. Binary strings — TEXT fields that store raw bytes (e.g.
// RawInstanceAddress.Unpack, some address fields) — are hex-encoded so they render as readable hex.
// Nested structs become map[string]any (preserving field names via  JSON tags); slices become []any.
// The CLDF renderer handles these types via getFieldValue/YamlField.
func toDisplayArg(rv reflect.Value) any {
	// Dereference pointers; nil pointer → nil.
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.String:
		s := rv.String()
		// TEXT/PARTY/CONTRACT_ID/NUMERIC are all string kinds. Some fields (e.g.
		// RawInstanceAddress.Unpack) store raw bytes. Hex-encode those so the renderer
		// shows "0x..." instead of binary characters.
		if !utf8.ValidString(s) || containsControlBytes(s) {
			return "0x" + hex.EncodeToString([]byte(s))
		}

		return s

	case reflect.Bool:
		return rv.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint()

	case reflect.Float32, reflect.Float64:
		return rv.Float()

	case reflect.Slice:
		if rv.IsNil() {
			return []any{}
		}
		result := make([]any, rv.Len())
		for i := range rv.Len() {
			result[i] = toDisplayArg(rv.Index(i))
		}

		return result

	case reflect.Map:
		result := make(map[string]any, rv.Len())
		for _, k := range rv.MapKeys() {
			result[fmt.Sprintf("%v", k.Interface())] = toDisplayArg(rv.MapIndex(k))
		}

		return result

	case reflect.Struct:
		// Recurse so nested Daml records become map[string]any with the same binary-string fix.
		rt := rv.Type()
		result := make(map[string]any, rt.NumField())
		for i := range rt.NumField() {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			name := f.Name
			if tag := f.Tag.Get("json"); tag != "" && tag != "-" {
				name = strings.Split(tag, ",")[0]
			}
			result[name] = toDisplayArg(rv.Field(i))
		}

		return result

	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128,
		reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Pointer, reflect.UnsafePointer:
		return fmt.Sprintf("%v", rv.Interface())
	}

	return fmt.Sprintf("%v", rv.Interface())
}

// containsControlBytes reports whether s contains control characters (below U+0020) other than
// the common whitespace \t \n \r. Used to detect strings that carry raw binary bytes rather than
// human-readable text.
func containsControlBytes(s string) bool {
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
		if !unicode.IsPrint(r) && r != '\t' && r != '\n' && r != '\r' {
			return true
		}
	}

	return false
}
