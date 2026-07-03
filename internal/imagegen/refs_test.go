package imagegen

import (
	"testing"
)

func TestDetectMIME(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "png",
			data: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0},
			want: "image/png",
		},
		{
			name: "jpeg",
			data: []byte{0xff, 0xd8, 0xff, 0xe0},
			want: "image/jpeg",
		},
		{
			name: "webp",
			data: append([]byte("RIFF"), append([]byte{0, 0, 0, 0}, []byte("WEBP")...)...),
			want: "image/webp",
		},
		{
			name: "unknown",
			data: []byte{0x00, 0x01, 0x02, 0x03},
			want: "",
		},
		{
			name: "too_short",
			data: []byte{0x89},
			want: "",
		},
		{
			name: "gif_rejected",
			data: []byte("GIF89a"),
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectMIME(tc.data)
			if got != tc.want {
				t.Errorf("detectMIME() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEnforceRefCap(t *testing.T) {
	makeRefs := func(n int) []*RefImage {
		refs := make([]*RefImage, n)
		for i := range refs {
			refs[i] = &RefImage{Label: string(rune('A' + i))}
		}
		return refs
	}

	t.Run("under_cap", func(t *testing.T) {
		refs := makeRefs(3)
		kept, dropped := EnforceRefCap(refs)
		if len(kept) != 3 {
			t.Errorf("kept = %d, want 3", len(kept))
		}
		if len(dropped) != 0 {
			t.Errorf("dropped = %v, want none", dropped)
		}
	})

	t.Run("at_cap", func(t *testing.T) {
		refs := makeRefs(4)
		kept, dropped := EnforceRefCap(refs)
		if len(kept) != 4 {
			t.Errorf("kept = %d, want 4", len(kept))
		}
		if len(dropped) != 0 {
			t.Errorf("dropped = %v, want none", dropped)
		}
	})

	t.Run("over_cap", func(t *testing.T) {
		refs := makeRefs(6)
		kept, dropped := EnforceRefCap(refs)
		if len(kept) != 4 {
			t.Errorf("kept = %d, want 4", len(kept))
		}
		if len(dropped) != 2 {
			t.Errorf("dropped = %d, want 2", len(dropped))
		}
		if dropped[0] != "E" || dropped[1] != "F" {
			t.Errorf("dropped = %v, want [E F]", dropped)
		}
	})
}

func TestExpandTilde(t *testing.T) {
	got := expandTilde("/absolute/path")
	if got != "/absolute/path" {
		t.Errorf("absolute path changed: %q", got)
	}

	got = expandTilde("relative/path")
	if got != "relative/path" {
		t.Errorf("relative path changed: %q", got)
	}

	// tilde expansion
	got = expandTilde("~/test")
	if got == "~/test" {
		t.Error("tilde was not expanded")
	}
}
