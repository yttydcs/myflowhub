package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

type variableDomainCodecFixture struct {
	ArbitraryName NodeID            `json:"arbitrary_name"`
	Nodes         []NodeID          `json:"nodes"`
	Lookup        map[string]NodeID `json:"lookup"`
	Optional      *NodeID           `json:"optional,omitempty"`
	Text          string            `json:"text"`
	Signed        int64             `json:"signed"`
	Unsigned      uint64            `json:"unsigned"`
	Plain         uint64            `json:"plain"`
	Reject        bool              `json:"reject,omitempty"`
}

func (v variableDomainCodecFixture) Validate() error {
	if v.Reject {
		return errors.New("owner business rule rejected payload")
	}
	return nil
}

func variableDomainCodecDefinition(t *testing.T) DataSchemaDefinition {
	t.Helper()
	definition, err := dataSchemaFromTypeMode(reflect.TypeFor[variableDomainCodecFixture](), 1, true)
	if err != nil {
		t.Fatal(err)
	}
	*dataSchemaAtPath(&definition, "signed") = DataSchemaDefinition{Type: "string", Format: "int64"}
	*dataSchemaAtPath(&definition, "unsigned") = DataSchemaDefinition{Type: "string", Format: "uint64"}
	// Some existing domain fields already use decimal strings. Decoding must
	// leave those strings intact even when their definition supplies a format.
	dataSchemaAtPath(&definition, "text").Format = "node-id"
	if err := ValidateVariableDefinition(definition); err != nil {
		t.Fatal(err)
	}
	return definition
}

func variableDomainCodecValue() variableDomainCodecFixture {
	owner := NodeID(math.MaxUint64)
	return variableDomainCodecFixture{
		ArbitraryName: owner, Nodes: []NodeID{1, 9007199254740993, owner},
		Lookup: map[string]NodeID{"root": owner}, Optional: &owner,
		Text: "18446744073709551615", Signed: math.MinInt64, Unsigned: math.MaxUint64,
		Plain: MaxVariableSafeInteger,
	}
}

func TestVariableDomainCodecFullWidthRoundTrip(t *testing.T) {
	definition := variableDomainCodecDefinition(t)
	original := variableDomainCodecValue()
	data, err := EncodeVariablePayload(definition, original, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`"arbitrary_name":"18446744073709551615"`, `"nodes":["1","9007199254740993","18446744073709551615"]`,
		`"lookup":{"root":"18446744073709551615"}`, `"optional":"18446744073709551615"`,
		`"text":"18446744073709551615"`, `"signed":"-9223372036854775808"`,
		`"unsigned":"18446744073709551615"`, `"plain":9007199254740991`,
	} {
		if !bytes.Contains(data, []byte(required)) {
			t.Fatalf("missing exact wire %s in %s", required, data)
		}
	}
	if err := ValidateVariableValue(definition, data); err != nil {
		t.Fatal(err)
	}
	var decoded variableDomainCodecFixture
	if err := DecodeVariablePayload(definition, data, DefaultMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("round-trip lost exact domain values: %#v", decoded)
	}

	original.Optional, original.Nodes, original.Lookup = nil, nil, nil
	data, err = EncodeVariablePayload(definition, original, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(`"optional":`)) || !bytes.Contains(data, []byte(`"nodes":null`)) || !bytes.Contains(data, []byte(`"lookup":null`)) {
		t.Fatalf("nil/absent presence changed: %s", data)
	}
	if err := DecodeVariablePayload(definition, data, DefaultMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("null/absent decode differs: %#v", decoded)
	}
}

func TestVariableDomainCodecRejectsNoncanonicalOrOldNumbers(t *testing.T) {
	definition := variableDomainCodecDefinition(t)
	original := variableDomainCodecValue()
	data, err := EncodeVariablePayload(definition, original, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, field, raw string }{
		{"old-node-number", "arbitrary_name", `18446744073709551615`},
		{"old-small-node-number", "arbitrary_name", `1`},
		{"node-zero", "arbitrary_name", `"0"`},
		{"node-leading-zero", "arbitrary_name", `"01"`},
		{"node-plus", "arbitrary_name", `"+1"`},
		{"node-exponent", "arbitrary_name", `"1e1"`},
		{"node-overflow", "arbitrary_name", `"18446744073709551616"`},
		{"node-null", "arbitrary_name", `null`},
		{"old-int64-number", "signed", `-9223372036854775808`},
		{"int64-underflow", "signed", `"-9223372036854775809"`},
		{"int64-overflow", "signed", `"9223372036854775808"`},
		{"int64-negative-zero", "signed", `"-0"`},
		{"uint64-negative", "unsigned", `"-1"`},
		{"uint64-overflow", "unsigned", `"18446744073709551616"`},
		{"uint64-noncanonical", "unsigned", `"00"`},
		{"plain-unsafe-integer", "plain", `9007199254740992`},
		{"plain-unrequested-string", "plain", `"1"`},
		{"old-node-in-array", "nodes", `[1]`},
		{"old-node-in-map", "lookup", `{"root":1}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			fields[test.field] = json.RawMessage(test.raw)
			bad, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			var decoded variableDomainCodecFixture
			if err := DecodeVariablePayload(definition, bad, DefaultMaxPayload, &decoded); err == nil {
				t.Fatalf("accepted invalid current-format value %s", bad)
			}
		})
	}
	old, err := EncodeJSONPayload(original, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded variableDomainCodecFixture
	if err := DecodeVariablePayload(definition, old, DefaultMaxPayload, &decoded); err == nil {
		t.Fatal("accepted unconverted old domain JSON")
	}
	duplicate := strings.Replace(string(data), `"plain":9007199254740991`, `"plain":1,"plain":2`, 1)
	if err := DecodeVariablePayload(definition, json.RawMessage(duplicate), DefaultMaxPayload, &decoded); err == nil {
		t.Fatal("accepted duplicate keys")
	}
}

func TestVariableDomainCodecBoundsAndOwnerValidation(t *testing.T) {
	definition := variableDomainCodecDefinition(t)
	for _, limits := range []struct {
		signed   int64
		unsigned uint64
	}{{math.MinInt64, 0}, {math.MaxInt64, math.MaxUint64}, {0, 0}} {
		original := variableDomainCodecValue()
		original.Signed, original.Unsigned = limits.signed, limits.unsigned
		data, err := EncodeVariablePayload(definition, original, DefaultMaxPayload)
		if err != nil {
			t.Fatal(err)
		}
		var decoded variableDomainCodecFixture
		if err := DecodeVariablePayload(definition, data, DefaultMaxPayload, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("decimal bounds changed: %#v", decoded)
		}
	}
	original := variableDomainCodecValue()
	original.Plain = MaxVariableSafeInteger + 1
	if _, err := EncodeVariablePayload(definition, original, DefaultMaxPayload); err == nil {
		t.Fatal("ordinary integer silently converted or precision weakened")
	}
	original = variableDomainCodecValue()
	original.Reject = true
	if _, err := EncodeVariablePayload(definition, original, DefaultMaxPayload); err == nil {
		t.Fatal("encode ignored domain Validate")
	}
	original.Reject = false
	data, err := EncodeVariablePayload(definition, original, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	bad := strings.TrimSuffix(string(data), "}") + `,"reject":true}`
	var decoded variableDomainCodecFixture
	if err := DecodeVariablePayload(definition, json.RawMessage(bad), DefaultMaxPayload, &decoded); err == nil {
		t.Fatal("decode ignored domain Validate")
	}
	if _, err := EncodeVariablePayload(definition, original, len(data)-1); err == nil {
		t.Fatal("encode ignored configured payload size")
	}
	if err := DecodeVariablePayload(definition, data, len(data)-1, &decoded); err == nil {
		t.Fatal("decode ignored configured payload size")
	}
}

func TestVariableNodeIDViewAndNonVariableWireUnchanged(t *testing.T) {
	before, err := MarshalBuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	definition, err := DataSchemaByID(SchemaResourceCatalogV2)
	if err != nil {
		t.Fatal(err)
	}
	resources := dataSchemaAtPath(&definition, "resources")
	owner := dataSchemaAtPath(resources.Items, "id.owner")
	if owner.Type != "string" || owner.Format != "node-id" || owner.Minimum != nil || owner.Maximum != nil || owner.MultipleOf != nil {
		t.Fatalf("wrong variable owner view: %#v", owner)
	}
	definitions, err := BuiltinDataSchemas()
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range definitions {
		variable, err := DataSchemaByID(domain.ID)
		if err != nil {
			t.Fatal(err)
		}
		if domain.ID != SchemaResourceCatalogV2 && !reflect.DeepEqual(domain, variable) {
			t.Fatalf("unexpected NodeID projection in %s", domain.ID)
		}
		if domain.ID == SchemaResourceCatalogV2 {
			field := dataSchemaAtPath(dataSchemaAtPath(&domain, "resources").Items, "id.owner")
			if field.Type != "integer" || field.Format != "" {
				t.Fatalf("ordinary domain definition changed to variable string: %#v", field)
			}
		}
	}
	after, err := MarshalBuiltinDataSchemas()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("variable lookup mutated ordinary builtin definitions")
	}
	id := ResourceID{Owner: NodeID(math.MaxUint64), Name: "example"}
	ordinary, err := json.Marshal(id)
	if err != nil || string(ordinary) != `{"owner":18446744073709551615,"name":"example"}` {
		t.Fatalf("global NodeID JSON changed: %s, %v", ordinary, err)
	}
	fixture := variableDomainCodecValue()
	ordinary, err = EncodeJSONPayload(fixture, DefaultMaxPayload)
	if err != nil || !bytes.Contains(ordinary, []byte(`"arbitrary_name":18446744073709551615`)) {
		t.Fatalf("non-variable encoding changed: %s, %v", ordinary, err)
	}
	// Numeric-looking string names and arbitrary uint64 fields are not a
	// signal to infer a NodeID. Only the exact Go NodeID type is projected.
	type shape struct {
		Unrelated NodeID `json:"unrelated"`
		NodeID    uint64 `json:"node_id"`
		Owner     string `json:"owner"`
	}
	view, err := dataSchemaFromTypeMode(reflect.TypeFor[shape](), 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if dataSchemaAtPath(&view, "unrelated").Format != "node-id" || dataSchemaAtPath(&view, "node_id").Type != "integer" || dataSchemaAtPath(&view, "owner").Format != "" {
		t.Fatalf("variable reflection guessed fields by name: %#v", view)
	}
}

type variableDomainRawFixture struct{ raw json.RawMessage }

func (v variableDomainRawFixture) Validate() error              { return nil }
func (v variableDomainRawFixture) MarshalJSON() ([]byte, error) { return v.raw, nil }

func TestVariableDomainCodecRejectsInvalidDomainJSON(t *testing.T) {
	definition := DataSchemaDefinition{Type: "object", Properties: []DataSchemaProperty{
		{Name: "text", Required: true, Schema: &DataSchemaDefinition{Type: "string"}},
	}}
	for _, raw := range []string{`{"text":"first","text":"second"}`, `{"text":"\ud800"}`, `{"text":"\udfff"}`} {
		t.Run(raw, func(t *testing.T) {
			if encoded, err := EncodeVariablePayload(definition, variableDomainRawFixture{json.RawMessage(raw)}, DefaultMaxPayload); err == nil {
				t.Fatalf("invalid domain JSON was silently normalized: %s -> %s", raw, encoded)
			}
		})
	}
}

func TestVariableDomainCodecInvalidTargetsAndFailureAtomicity(t *testing.T) {
	definition := variableDomainCodecDefinition(t)
	value := variableDomainCodecValue()
	data, err := EncodeVariablePayload(definition, value, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	var typedNil *variableDomainCodecFixture
	for _, target := range []ValidatedPayload{nil, typedNil, value} {
		if err := DecodeVariablePayload(definition, data, DefaultMaxPayload, target); err == nil {
			t.Fatalf("accepted invalid decode destination %T", target)
		}
	}
	for _, raw := range []string{
		strings.TrimSuffix(string(data), "}") + `,"reject":true}`,
		strings.Replace(string(data), `"plain":9007199254740991`, `"plain":9007199254740992`, 1),
	} {
		destination := variableDomainCodecValue()
		destination.Text = "1"
		original := destination
		if err := DecodeVariablePayload(definition, json.RawMessage(raw), DefaultMaxPayload, &destination); err == nil {
			t.Fatal("invalid payload accepted")
		}
		if !reflect.DeepEqual(destination, original) {
			t.Fatalf("failed decode overwrote caller state: %#v", destination)
		}
	}
	for _, source := range []ValidatedPayload{nil, typedNil} {
		t.Run("nil-source", func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("nil source must return an error, panicked: %v", recovered)
				}
			}()
			if _, err := EncodeVariablePayload(definition, source, DefaultMaxPayload); err == nil {
				t.Fatalf("accepted nil encode source %T", source)
			}
		})
	}
}

type variableDomainQuotedFixture struct {
	Owner NodeID `json:"owner,string"`
	Text  string `json:"text"`
}

func (v variableDomainQuotedFixture) Validate() error { return nil }

func TestVariableDomainCodecPreservesExistingQuotedFields(t *testing.T) {
	definition := DataSchemaDefinition{Type: "object", Properties: []DataSchemaProperty{
		{Name: "owner", Required: true, Schema: &DataSchemaDefinition{Type: "string", Format: "node-id"}},
		{Name: "text", Required: true, Schema: &DataSchemaDefinition{Type: "string", Format: "uint64"}},
	}}
	value := variableDomainQuotedFixture{Owner: NodeID(math.MaxUint64), Text: "18446744073709551615"}
	data, err := EncodeVariablePayload(definition, value, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded variableDomainQuotedFixture
	if err := DecodeVariablePayload(definition, data, DefaultMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != value {
		t.Fatalf("existing string/quoted wire changed: %#v", decoded)
	}
}

func TestVariableDomainCodecCatalogAndExistingProviderStrings(t *testing.T) {
	owner := NodeID(math.MaxUint64)
	catalog := ResourceCatalogV2{Version: 2, Revision: 1, Resources: []ResourceDescriptorV2{{
		ID: ResourceID{Owner: owner, Name: BuiltinResourceCatalog}, Type: ResourceTypeVariable, TypeVersion: 2,
		Capabilities: []CapabilityDescriptorV2{{Name: CapabilityRead, Permission: "catalog.read", OutputSchema: SchemaVariableValueV1, MaxPayloadBytes: DefaultMaxPayload}},
		Schemas:      []SchemaDescriptorV2{{ID: SchemaVariableValueV1, ContentType: "application/json"}},
		Limits:       ResourceLimitsV2{MaxPayloadBytes: DefaultMaxPayload},
	}}}
	definition, err := DataSchemaByID(SchemaResourceCatalogV2)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeVariablePayload(definition, catalog, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ResourceCatalogV2
	if err := DecodeVariablePayload(definition, encoded, DefaultMaxPayload, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(catalog, decoded) {
		t.Fatalf("catalog domain round-trip differs: %#v", decoded)
	}
	ordinary, err := EncodeJSONPayload(catalog, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(ordinary, []byte(`"owner":18446744073709551615`)) || !bytes.Contains(encoded, []byte(`"owner":"18446744073709551615"`)) {
		t.Fatalf("catalog wire boundary not isolated: %s / %s", ordinary, encoded)
	}
	if err := DecodeVariablePayload(definition, ordinary, DefaultMaxPayload, &decoded); err == nil {
		t.Fatal("catalog accepted old numeric owner")
	}

	transfers := FileTransfersV1{Version: 1, Revision: 1, Transfers: []FileTransferSummaryV1{{
		TransferID: strings.Repeat("f", 32), OwnerNodeID: "18446744073709551615", Path: "example.bin",
		TotalBytes: 2, State: "offered", ExpiresAtUnixMS: 1,
	}}}
	definition, err = DataSchemaByID(SchemaFileTransfersV1)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err = EncodeVariablePayload(definition, transfers, DefaultMaxPayload)
	if err != nil {
		t.Fatal(err)
	}
	var result FileTransfersV1
	if err := DecodeVariablePayload(definition, encoded, DefaultMaxPayload, &result); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(transfers, result) {
		t.Fatalf("existing string owner changed: %#v", result)
	}
}

type variableDomainBinaryFixture struct {
	Data []byte `json:"data"`
}

func (v variableDomainBinaryFixture) Validate() error { return nil }

func TestVariableDomainCodecBinaryRemainsBase64(t *testing.T) {
	definition, err := dataSchemaFromTypeMode(reflect.TypeFor[variableDomainBinaryFixture](), 1, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{nil, {}, {0, 255, 128}} {
		original := variableDomainBinaryFixture{Data: data}
		encoded, err := EncodeVariablePayload(definition, original, DefaultMaxPayload)
		if err != nil {
			t.Fatal(err)
		}
		var decoded variableDomainBinaryFixture
		if err := DecodeVariablePayload(definition, encoded, DefaultMaxPayload, &decoded); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("binary/null/empty round-trip changed: %#v", decoded)
		}
	}
}
