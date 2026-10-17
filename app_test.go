package main

import (
	"reflect"
	"testing"
)

// Wails only binds this surface; persistence/DPAPI helpers must stay private.
func TestFrontendBoundary(t *testing.T) {
	allowed := map[string]bool{"Configuration": true, "Configure": true, "List": true, "Get": true, "Create": true, "Update": true, "Delete": true, "SetSessionState": true}
	typ := reflect.TypeOf(&App{})
	for i := 0; i < typ.NumMethod(); i++ {
		if !allowed[typ.Method(i).Name] {
			t.Fatalf("unexpected frontend method: %s", typ.Method(i).Name)
		}
	}
	config, _ := typ.MethodByName("Configuration")
	dto := config.Type.Out(0)
	for i := 0; i < dto.NumField(); i++ {
		name := dto.Field(i).Name
		if name != "URL" && name != "Configured" && name != "Problem" {
			t.Fatalf("unexpected public config field: %s", name)
		}
	}
}
