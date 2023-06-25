package test

import (
	"fmt"
	"testing"

	uuid "github.com/satori/go.uuid"
)

func TestGenerateUUID(t *testing.T) {
	v4 := uuid.NewV4()
	fmt.Println(v4)
}
