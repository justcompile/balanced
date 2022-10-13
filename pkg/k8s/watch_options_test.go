package k8s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_WithResyncInterval(t *testing.T) {
	w := &Watcher{}

	WithResyncInterval(time.Hour)(w)

	expected := time.Hour

	assert.Equal(t, &expected, w.resyncInterval)
}
