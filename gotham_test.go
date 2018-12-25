package gotham

import "testing"
import "runtime"
import "github.com/stretchr/testify/assert"

func TestDefault(t *testing.T) {
	cfg := Default()
	assert.Equal(t, cfg.Port, 8202, "default port should be 8202.")
	assert.Equal(t, cfg.Network, "tcp", "default protocol should be tcp.")
	assert.Equal(t, cfg.ReusePort, true, "default reuseport should be true.")
	assert.Equal(t, cfg.Stdlib, false, "default stdlib for networking should be false.")
	assert.Equal(t, cfg.NumLoops, runtime.NumCPU(), "default stdlib for networking should be false.")

	assert.Equal(t, cfg.getAddr(), "tcp://localhost:8202?reusePort=true")
}
