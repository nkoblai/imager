package router

import (
	"testing"
)

func TestNew(t *testing.T) {
	if New(nil, nil, nil) == nil {
		t.Fatal("calling to New shouldn't return nil")
	}
}
