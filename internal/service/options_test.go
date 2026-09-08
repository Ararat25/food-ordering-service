package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func TestValidateOptionSelection(t *testing.T) {
	definitions := json.RawMessage(`[{"name":"size","options":["small","medium","large"],"default":"medium"}]`)
	tests := []struct {
		name     string
		selected json.RawMessage
		want     string
		wantErr  bool
	}{
		{name: "fills default", want: `{"size":"medium"}`},
		{name: "accepts allowed value", selected: json.RawMessage(`{"size":"large"}`), want: `{"size":"large"}`},
		{name: "rejects unknown value", selected: json.RawMessage(`{"size":"giant"}`), wantErr: true},
		{name: "rejects unknown option", selected: json.RawMessage(`{"crust":"thin"}`), wantErr: true},
		{name: "rejects non-object selection", selected: json.RawMessage(`"large"`), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateOptionSelection(definitions, tt.selected)
			if tt.wantErr {
				require.ErrorIs(t, err, domain.ErrValidation)
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestParseMenuOptionsRejectsInvalidDefinitions(t *testing.T) {
	for _, raw := range []string{
		`{"name":"size"}`,
		`[{"name":"size","options":[]}]`,
		`[{"name":"size","options":["small"],"default":"large"}]`,
		`[{"name":"size","options":["small"]},{"name":"size","options":["large"]}]`,
	} {
		_, err := parseMenuOptions(json.RawMessage(raw))
		require.ErrorIs(t, err, domain.ErrValidation, "raw: %s", raw)
	}
}

func TestSameOptionsIgnoresObjectKeyOrder(t *testing.T) {
	require.True(t, sameOptions(json.RawMessage(`{"size":"large","crust":"thin"}`), json.RawMessage(`{"crust":"thin","size":"large"}`)))
}
