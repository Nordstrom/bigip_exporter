package main

import "testing"

func TestSomeTest(t *testing.T)  {
	t.Logf("Running test: %s", t.Name())
}
