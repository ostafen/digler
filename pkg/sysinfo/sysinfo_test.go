package sysinfo_test

import (
	"testing"

	"github.com/ostafen/digler/pkg/sysinfo"
)

func TestListDevices(t *testing.T) {
	devices, err := sysinfo.ListDevices()
	if err != nil {
		t.Fatalf("ListDevices returned error: %v", err)
	}

	t.Logf("ListDevices found %d devices", len(devices))
	for i, dev := range devices {
		t.Logf("[%d] Name: %q, Path: %q, Size: %d, Model: %q, Removable: %v",
			i, dev.Name, dev.Path, dev.Size, dev.Model, dev.Removable)
		if dev.Path == "" {
			t.Errorf("device %d has empty path", i)
		}
	}
}
