package ui

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	background = func(work func()) { work() }
	os.Exit(m.Run())
}
