package main

import (
	"testing"

	"gitops.szakallas.eu/plugins/internal/util"
)

func TestYQTransform(t *testing.T) {
	util.TestE2E(t, "../../tests/yqtransform")
}
