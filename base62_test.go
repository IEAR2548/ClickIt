package main

import "testing"

func TestEncodeBase62(t *testing.T) {
	cases := []struct {
		name string
		id   int64
		want string
	}{
		{name: "zero", id: 0, want: "1Luue"},
		{name: "single digit", id: 1, want: "1Luuf"},
		{name: "first letter boundary", id: 10, want: "1Luuo"},
		{name: "two characters", id: 125, want: "1Luwf"},
		{name: "lowercase boundary", id: 36, want: "1LuvE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EncodeBase62(tc.id)
			if got != tc.want {
				t.Errorf("EncodeBase62(%d) = %q, want %q", tc.id, got, tc.want)
			}
		})
	}
}
