package settings

import (
	"errors"
	"os"
	"os/exec"
	"testing"
)

func TestWriteReplaceAndInvalidInputs(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Set("quota", 3); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("quota", 7); err != nil {
		t.Fatal(err)
	}
	if err := store.Set("../escape", 9); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if err := store.Set("quota", -1); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if value, err := store.Get("quota"); err != nil || value != 7 {
		t.Fatalf("value=%d err=%v", value, err)
	}
	if _, err := store.Get("absent"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestStoredValueSurvivesWriterProcessExit(t *testing.T) {
	if os.Getenv("SETTINGS_DEMO_CHILD") == "write" {
		store, err := Open(os.Getenv("SETTINGS_DEMO_DIRECTORY"))
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Set("durable", 42); err != nil {
			t.Fatal(err)
		}
		return
	}
	directory := t.TempDir()
	command := exec.Command(os.Args[0], "-test.run=^TestStoredValueSurvivesWriterProcessExit$")
	command.Env = append(os.Environ(), "SETTINGS_DEMO_CHILD=write", "SETTINGS_DEMO_DIRECTORY="+directory)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("writer process: %v %s", err, output)
	}
	store, err := Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	if value, err := store.Get("durable"); err != nil || value != 42 {
		t.Fatalf("persisted value=%d err=%v", value, err)
	}
}
