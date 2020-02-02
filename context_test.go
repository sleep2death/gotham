package gotham

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextReset(t *testing.T) {
	router := New()
	c := router.allocateContext()
	assert.Equal(t, c.router, router)

	c.index = 2
	c.Writer = &ResponseWriter{}
}
