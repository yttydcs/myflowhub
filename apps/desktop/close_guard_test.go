package main

import (
	"errors"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func TestUnsavedCloseRequiresExplicitConfirmation(t *testing.T) {
	if preventUnsavedClose(false, func() (bool, error) { t.Fatal("clean exit prompted"); return false, nil }) {
		t.Fatal("clean exit blocked")
	}
	if !preventUnsavedClose(true, func() (bool, error) { return false, nil }) {
		t.Fatal("cancel did not preserve edits")
	}
	if preventUnsavedClose(true, func() (bool, error) { return true, nil }) {
		t.Fatal("explicit discard blocked")
	}
	if !preventUnsavedClose(true, func() (bool, error) { return false, errors.New("dialog failed") }) {
		t.Fatal("failed confirmation discarded edits")
	}
}

func TestUnsavedCloseUsesNativeQuestionResponsesAndSafeDefault(t *testing.T) {
	options := unsavedCloseDialog()
	if options.Type != runtime.QuestionDialog || options.DefaultButton != "No" || options.CancelButton != "No" || len(options.Buttons) != 2 || options.Buttons[0] != "Yes" || options.Buttons[1] != "No" {
		t.Fatalf("unexpected native close confirmation: %+v", options)
	}
}
