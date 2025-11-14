package main

import (
	"testing"

	"gitops.szakallas.eu/plugins/internal/utils"
)

func TestYQTransform(t *testing.T) {
	utils.TestE2E(t, "../../tests/yqtransform")
}
