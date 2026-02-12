package canton

import (
	"testing"

	apiv2 "github.com/digital-asset/dazl-client/v8/go/api/com/daml/ledger/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTemplateIDFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		templateID string
		wantPkg    string
		wantModule string
		wantEntity string
		wantErr    string
	}{
		{
			name:       "valid template ID",
			templateID: "#packageid123:MCMS.Main:MCMS",
			wantPkg:    "#packageid123",
			wantModule: "MCMS.Main",
			wantEntity: "MCMS",
			wantErr:    "",
		},
		{
			name:       "another valid template ID",
			templateID: "#abc123def456:Module.Submodule:Contract",
			wantPkg:    "#abc123def456",
			wantModule: "Module.Submodule",
			wantEntity: "Contract",
			wantErr:    "",
		},
		{
			name:       "missing hash prefix",
			templateID: "packageid123:MCMS.Main:MCMS",
			wantPkg:    "",
			wantModule: "",
			wantEntity: "",
			wantErr:    "template ID must start with #",
		},
		{
			name:       "too few parts",
			templateID: "#packageid123:MCMS",
			wantPkg:    "",
			wantModule: "",
			wantEntity: "",
			wantErr:    "template ID must have format #package:module:entity",
		},
		{
			name:       "too many parts",
			templateID: "#packageid123:MCMS.Main:MCMS:Extra",
			wantPkg:    "",
			wantModule: "",
			wantEntity: "",
			wantErr:    "template ID must have format #package:module:entity",
		},
		{
			name:       "empty string",
			templateID: "",
			wantPkg:    "",
			wantModule: "",
			wantEntity: "",
			wantErr:    "template ID must start with #",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pkg, module, entity, err := ParseTemplateIDFromString(tt.templateID)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Empty(t, pkg)
				assert.Empty(t, module)
				assert.Empty(t, entity)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantPkg, pkg)
				assert.Equal(t, tt.wantModule, module)
				assert.Equal(t, tt.wantEntity, entity)
			}
		})
	}
}

func TestFormatTemplateID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   *apiv2.Identifier
		want string
	}{
		{
			name: "valid identifier",
			id: &apiv2.Identifier{
				PackageId:  "packageid123",
				ModuleName: "MCMS.Main",
				EntityName: "MCMS",
			},
			want: "packageid123:MCMS.Main:MCMS",
		},
		{
			name: "another valid identifier",
			id: &apiv2.Identifier{
				PackageId:  "abc123def456",
				ModuleName: "Module.Submodule",
				EntityName: "Contract",
			},
			want: "abc123def456:Module.Submodule:Contract",
		},
		{
			name: "nil identifier",
			id:   nil,
			want: "",
		},
		{
			name: "empty fields",
			id: &apiv2.Identifier{
				PackageId:  "",
				ModuleName: "",
				EntityName: "",
			},
			want: "::",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FormatTemplateID(tt.id)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeTemplateKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tid  string
		want string
	}{
		{
			name: "with hash prefix and full path",
			tid:  "#packageid123:MCMS.Main:MCMS",
			want: "MCMS.Main:MCMS",
		},
		{
			name: "without hash prefix",
			tid:  "packageid123:MCMS.Main:MCMS",
			want: "MCMS.Main:MCMS",
		},
		{
			name: "only two parts",
			tid:  "MCMS.Main:MCMS",
			want: "MCMS.Main:MCMS",
		},
		{
			name: "single part",
			tid:  "MCMS",
			want: "MCMS",
		},
		{
			name: "four parts with hash",
			tid:  "#pkg:ver:Module:Entity",
			want: "Module:Entity",
		},
		{
			name: "empty string",
			tid:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NormalizeTemplateKey(tt.tid)
			assert.Equal(t, tt.want, got)
		})
	}
}
