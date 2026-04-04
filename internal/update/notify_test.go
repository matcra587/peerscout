package update_test

import (
	"testing"

	"github.com/matcra587/peerscout/internal/update"
	"github.com/stretchr/testify/assert"
)

func TestShouldCheck(t *testing.T) {
	tests := []struct {
		name   string
		envVar string
		agent  bool
		tty    bool
		want   bool
	}{
		{"normal TTY", "", false, true, true},
		{"agent mode", "", true, true, false},
		{"not TTY", "", false, false, false},
		{"env disabled", "1", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVar != "" {
				t.Setenv("PEERSCOUT_NO_UPDATE_CHECK", tt.envVar)
			} else {
				t.Setenv("PEERSCOUT_NO_UPDATE_CHECK", "")
			}
			assert.Equal(t, tt.want, update.ShouldCheck(tt.agent, tt.tty))
		})
	}
}
