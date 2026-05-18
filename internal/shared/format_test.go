package shared

import "testing"

func TestFormatMeteringPoint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "33-char canonical: AT + 11 digits + 20 digits",
			in:   "AT0031000000000000000000990022105",
			want: "AT 003100 00000 00000000000990022105",
		},
		{
			name: "33-char with trailing letter in last block",
			in:   "AT003100000000000000000099002210X",
			want: "AT 003100 00000 0000000000099002210X",
		},
		{
			name: "32-char (too short) returned unchanged",
			in:   "AT00310000000000000000009900221",
			want: "AT00310000000000000000009900221",
		},
		{
			name: "34-char (too long) returned unchanged",
			in:   "AT001234567890123456789012345678901",
			want: "AT001234567890123456789012345678901",
		},
		{
			name: "non-AT prefix returned unchanged",
			in:   "DE0031000000000000000000990022105",
			want: "DE0031000000000000000000990022105",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatMeteringPoint(tc.in)
			if got != tc.want {
				t.Errorf("FormatMeteringPoint(%q)\n got:  %q\n want: %q", tc.in, got, tc.want)
			}
		})
	}
}
