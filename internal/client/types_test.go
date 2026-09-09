package client

import (
	"encoding/json"
	"testing"
)

func TestFlexIntUnmarshal(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{`12`, 12, false},
		{`"12"`, 12, false},
		{`"12.00"`, 12, false},
		{`12.9`, 12, false},
		{`"  7 "`, 7, false},
		{`""`, 0, false},
		{`null`, 0, false},
		{`0`, 0, false},
		{`"abc"`, 0, true},
	}
	for _, tc := range cases {
		var f FlexInt
		err := json.Unmarshal([]byte(tc.in), &f)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Unmarshal(%s): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("Unmarshal(%s): %v", tc.in, err)
			continue
		}
		if f.Int64() != tc.want {
			t.Errorf("Unmarshal(%s) = %d, want %d", tc.in, f.Int64(), tc.want)
		}
	}
}

func TestFlexIntInStruct(t *testing.T) {
	var v struct {
		ID FlexInt `json:"id"`
	}
	if err := json.Unmarshal([]byte(`{"id":"4711"}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.ID.Int64() != 4711 {
		t.Fatalf("got %d", v.ID.Int64())
	}
}
