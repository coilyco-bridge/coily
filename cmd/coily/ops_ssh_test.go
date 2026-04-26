package main

import "testing"

func TestValidateUnitName(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"", true},
		{"-foo.service", true},
		{"foo.service", false},
		{"foo@bar.service", false},
		{"my_unit-1.service", false},
		{"foo;rm -rf /", true},
		{"foo`whoami`", true},
		{"foo$bar", true},
		{"a/b", true},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			err := validateUnitName(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateUnitName(%q) err=%v, wantErr=%v", tc.in, err, tc.wantErr)
			}
		})
	}
}
