package main

import (
	"testing"

	"gitops.szakallas.eu/plugins/internal/testutils"
)

func TestYQTransform(t *testing.T) {
	testutils.TestE2E(t, "./.")
}
