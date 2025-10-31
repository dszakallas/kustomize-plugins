package main

import (
	"testing"

	"gitops.szakallas.eu/plugins/internal/util"
)

func TestResourceInjector(t *testing.T) {
	util.TestE2E(t, "../../tests/resourceinjector")
}
