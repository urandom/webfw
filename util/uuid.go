package util

import (
	"crypto/rand"
	"fmt"
)

// UUID generates a UUID-v4 string
func UUID() string {
	var u [16]byte

	rand.Read(u[:])

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		u[:4], u[4:6], u[6:8], u[8:10], u[10:])
}
