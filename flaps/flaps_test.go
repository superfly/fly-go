package flaps

import (
	"testing"
)

func TestSnakeCase(t *testing.T) {
	type testcase struct {
		name string
		in   string
		want string
	}

	cases := []testcase{
		{name: "case1", in: "fooBar", want: "foo_bar"},
		{name: "case2", in: appCreate.String(), want: "app_create"},
	}
	for _, tc := range cases {
		got := snakeCase(tc.in)
		if got != tc.want {
			t.Errorf("%s, got '%v', want '%v'", tc.name, got, tc.want)
		}
	}
}
