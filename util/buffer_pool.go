package util

import (
	"bytes"
	"sync"
)

type bPool struct {
	sync.Pool
}

// BufferPool is a utility variable that provides bytes.Buffer objects.
// It embeds a sync.Pool object and provides a helper GetBuffer method
// that returns a cast *bytes.Buffer.
var BufferPool bPool

// GetBuffer returns a bytes.Buffer pointer from the pool.
func (b *bPool) GetBuffer() *bytes.Buffer {
	buffer := b.Get().(*bytes.Buffer)
	buffer.Reset()

	return buffer
}

func init() {
	BufferPool = bPool{
		Pool: sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}
