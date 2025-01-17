package parser

import (
	"reflect"
	"testing"
)

func TestParser(t *testing.T) {
	p := NewParser()

	tests := []struct {
		input    string
		expected Command
		wantErr  bool
	}{
		{
			input: "SET key value",
			expected: Command{
				Type:  SET,
				Key:   "key",
				Value: "value",
			},
		},
		{
			input: "GET key",
			expected: Command{
				Type: GET,
				Key:  "key",
			},
		},
		{
			input:   "DEL",
			wantErr: true,
		},
		{
			input:   "UNKNOWN key",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		cmd, err := p.Parse(tt.input)
		if (err != nil) != tt.wantErr {
			t.Fatalf("input=%q, unexpected error status: got %v, wantErr=%v", tt.input, err, tt.wantErr)
		}
		if err == nil && !reflect.DeepEqual(cmd, tt.expected) {
			t.Errorf("input=%q, got %+v, want %+v", tt.input, cmd, tt.expected)
		}
	}
}
