package lark

import "testing"

func TestParseFreshSessionCommand(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantMatch bool
	}{
		{
			name:      "exact command matches",
			body:      "/new",
			wantMatch: true,
		},
		{
			name:      "leading and trailing whitespace tolerated",
			body:      "\n \t/new \t\n",
			wantMatch: true,
		},
		{
			name:      "same-line body rejected",
			body:      "/new start from scratch",
			wantMatch: false,
		},
		{
			name:      "multi-line body rejected",
			body:      "/new\nline one\nline two",
			wantMatch: false,
		},
		{
			name:      "prefix of token rejected",
			body:      "/newness is not a command",
			wantMatch: false,
		},
		{
			name:      "mid-sentence command rejected",
			body:      "please /new this run",
			wantMatch: false,
		},
		{
			name:      "wrong case rejected",
			body:      "/New help",
			wantMatch: false,
		},
		{
			name:      "normal body rejected",
			body:      "help me normally",
			wantMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd, ok := parseFreshSessionCommand(tc.body)
			if ok != tc.wantMatch {
				t.Fatalf("match=%v want %v (cmd=%+v)", ok, tc.wantMatch, cmd)
			}
			if !tc.wantMatch {
				if cmd != nil {
					t.Fatalf("expected nil command, got %+v", cmd)
				}
				return
			}
		})
	}
}
