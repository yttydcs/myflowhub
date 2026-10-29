package main

import (
	"errors"
	"testing"
)

func TestMetricsUnsavedCloseRequiresConfirmation(t *testing.T) {
	if preventUnsavedClose(false, func() (bool, error) { t.Fatal("clean close prompted"); return false, nil }) {
		t.Fatal("clean close blocked")
	}
	if !preventUnsavedClose(true, func() (bool, error) { return false, nil }) {
		t.Fatal("cancel lost edits")
	}
	if preventUnsavedClose(true, func() (bool, error) { return true, nil }) {
		t.Fatal("confirmed close blocked")
	}
	if !preventUnsavedClose(true, func() (bool, error) { return false, errors.New("dialog unavailable") }) {
		t.Fatal("dialog failure lost edits")
	}
}
