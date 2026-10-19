package plugin

import (
	"testing"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestResourceInputs_String covers the attribute-then-tag lookup.
//
// Attributes win when both sources set a property. Within one source the
// caller's key order wins. Empty and nil attributes fall through to tags.
// Numbers, strings, one-element lists, and a missing path are covered.
func TestResourceInputs_String(t *testing.T) {
	t.Parallel()

	withBoth, err := structpb.NewStruct(map[string]any{
		"storageType":   "io1",
		"size":          100.0,
		"empty":         "",
		"architectures": []any{"arm64"},
		"note":          "red hat",
	})
	require.NoError(t, err)

	tests := []struct {
		name string
		in   resourceInputs
		keys []string
		want string
		ok   bool
	}{
		{
			name: "attributes only",
			in:   resourceInputs{attrs: withBoth},
			keys: []string{"storageType"},
			want: "io1",
			ok:   true,
		},
		{
			name: "tags only",
			in:   resourceInputs{tags: map[string]string{"storage_type": "gp3"}},
			keys: []string{"storage_type"},
			want: "gp3",
			ok:   true,
		},
		{
			name: "both agree",
			in: resourceInputs{
				attrs: withBoth,
				tags:  map[string]string{"storageType": "io1"},
			},
			keys: []string{"storageType"},
			want: "io1",
			ok:   true,
		},
		{
			name: "attributes win over tags",
			in: resourceInputs{
				attrs: withBoth,
				tags:  map[string]string{"storageType": "gp2", "storage_type": "gp3"},
			},
			keys: []string{"storage_type", "storageType"},
			want: "io1",
			ok:   true,
		},
		{
			name: "canonical tag wins over alias tag",
			in: resourceInputs{
				tags: map[string]string{"storage_type": "gp3", "storageType": "io1"},
			},
			keys: []string{"storage_type", "storageType"},
			want: "gp3",
			ok:   true,
		},
		{
			name: "nil attributes use tags",
			in:   resourceInputs{tags: map[string]string{"size": "8"}},
			keys: []string{"size"},
			want: "8",
			ok:   true,
		},
		{
			name: "empty attributes use tags",
			in: resourceInputs{
				attrs: &structpb.Struct{},
				tags:  map[string]string{"size": "8"},
			},
			keys: []string{"size"},
			want: "8",
			ok:   true,
		},
		{
			name: "number becomes string",
			in:   resourceInputs{attrs: withBoth},
			keys: []string{"size"},
			want: "100",
			ok:   true,
		},
		{
			name: "string form",
			in:   resourceInputs{tags: map[string]string{"size": "20"}},
			keys: []string{"size"},
			want: "20",
			ok:   true,
		},
		{
			name: "one-element list",
			in:   resourceInputs{attrs: withBoth},
			keys: []string{"architectures"},
			want: "arm64",
			ok:   true,
		},
		{
			name: "value with a space",
			in:   resourceInputs{attrs: withBoth},
			keys: []string{"note"},
			want: "red hat",
			ok:   true,
		},
		{
			name: "missing path",
			in:   resourceInputs{attrs: withBoth, tags: map[string]string{"other": "x"}},
			keys: []string{"absent"},
			ok:   false,
		},
		{
			name: "empty tag is present",
			in:   resourceInputs{tags: map[string]string{"data_processed_gb": ""}},
			keys: []string{"data_processed_gb"},
			want: "",
			ok:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := tt.in.String(tt.keys...)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestResourceInputs_NumbersAndBool accepts number and string forms.
func TestResourceInputs_NumbersAndBool(t *testing.T) {
	t.Parallel()

	attrs, err := structpb.NewStruct(map[string]any{
		"allocatedStorage": 100.0,
		"memorySize":       "256",
		"multiAz":          true,
		"huge":             "9007199254740993",
	})
	require.NoError(t, err)
	in := resourceInputs{attrs: attrs, tags: map[string]string{"flag": "false"}}

	size, ok := in.Int("allocatedStorage")
	assert.True(t, ok)
	assert.Equal(t, 100, size)

	mem, ok := in.Int("memorySize")
	assert.True(t, ok)
	assert.Equal(t, 256, mem)

	huge, ok := in.Int("huge")
	assert.True(t, ok)
	assert.Equal(t, 9007199254740993, huge)

	multi, ok := in.Bool("multiAz")
	assert.True(t, ok)
	assert.True(t, multi)

	flag, ok := in.Bool("flag")
	assert.True(t, ok)
	assert.False(t, flag)
}

// TestResourceInputs_Struct reads a nested block and a one-element list.
func TestResourceInputs_Struct(t *testing.T) {
	t.Parallel()

	attrs, err := structpb.NewStruct(map[string]any{
		"rootBlockDevice": map[string]any{
			"volumeType": "gp3",
			"volumeSize": 20.0,
			"deviceName": "/dev/xvda extra",
		},
		"launchTemplate": []any{
			map[string]any{"id": "lt-123", "version": "$Latest"},
		},
	})
	require.NoError(t, err)
	in := newResourceInputs(&pbc.ResourceDescriptor{Attributes: attrs})

	root, ok := in.Struct("rootBlockDevice")
	require.True(t, ok)
	assert.Equal(t, "gp3", root.GetFields()["volumeType"].GetStringValue())
	assert.Equal(t, "/dev/xvda extra", root.GetFields()["deviceName"].GetStringValue())

	lt, ok := in.Struct("launchTemplate")
	require.True(t, ok)
	assert.Equal(t, "lt-123", lt.GetFields()["id"].GetStringValue())

	_, ok = in.Struct("missing")
	assert.False(t, ok)
}
