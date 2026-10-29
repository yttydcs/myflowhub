package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) SetUnsavedChanges(dirty bool) {
	a.mu.Lock()
	a.unsaved = dirty
	a.mu.Unlock()
}

func preventUnsavedClose(dirty bool, confirm func() (bool, error)) bool {
	if !dirty {
		return false
	}
	accepted, err := confirm()
	if err != nil {
		log.Printf("confirm unsaved close: %v", err)
		return true
	}
	return !accepted
}

func (a *App) BeforeClose(ctx context.Context) bool {
	a.mu.Lock()
	dirty := a.unsaved
	a.mu.Unlock()
	return preventUnsavedClose(dirty, func() (bool, error) {
		choice, err := runtime.MessageDialog(ctx, unsavedCloseDialog())
		return choice == "Yes", err
	})
}

func unsavedCloseDialog() runtime.MessageDialogOptions {
	// Windows maps QuestionDialog to native Yes/No and ignores custom labels.
	return runtime.MessageDialogOptions{
		Type: runtime.QuestionDialog, Title: "退出 MyFlowHub",
		Message: "视图或变量有未保存的更改。是否放弃这些更改并退出？选择“否”可继续编辑。",
		Buttons: []string{"Yes", "No"}, DefaultButton: "No", CancelButton: "No",
	}
}
