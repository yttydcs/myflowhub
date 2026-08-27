//go:build windows

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"unicode/utf16"

	"github.com/yttydcs/myflowhub/protocol"
)

func showSystemNotification(ctx context.Context, event protocol.NotificationEventV1) error {
	if ctx == nil {
		return errors.New("notification presenter context is required")
	}
	title, body := notificationText(event)
	command := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-WindowStyle", "Hidden", "-EncodedCommand", encodePowerShell(notificationScript(title, body)))
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message != "" {
			return fmt.Errorf("present Windows notification: %w: %s", err, message)
		}
		return fmt.Errorf("present Windows notification: %w", err)
	}
	return nil
}

func notificationText(event protocol.NotificationEventV1) (string, string) {
	title := strings.TrimSpace(event.Attributes["title"])
	if title == "" {
		title = "MyFlowHub / " + event.Channel
	}
	body := ""
	if strings.HasPrefix(event.ContentType, "text/") {
		body = strings.TrimSpace(string(event.Body))
	} else if event.ContentType == "application/json" {
		var object map[string]any
		if json.Unmarshal(event.Body, &object) == nil {
			for _, key := range []string{"body", "message", "text", "summary"} {
				if value, ok := object[key].(string); ok && strings.TrimSpace(value) != "" {
					body = strings.TrimSpace(value)
					break
				}
			}
		}
	}
	if body == "" {
		body = "New " + event.ContentType + " notification"
	}
	return truncateNotificationText(title, 96), truncateNotificationText(body, 220)
}

func truncateNotificationText(value string, maximum int) string {
	runes := []rune(strings.Join(strings.Fields(value), " "))
	if len(runes) <= maximum {
		return string(runes)
	}
	return string(runes[:maximum-3]) + "..."
}

func notificationScript(title, body string) string {
	title64 := base64.StdEncoding.EncodeToString([]byte(title))
	body64 := base64.StdEncoding.EncodeToString([]byte(body))
	return fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$title = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s'))
$body = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('%s'))
$notification = New-Object System.Windows.Forms.NotifyIcon
$notification.Icon = [System.Drawing.SystemIcons]::Information
$notification.BalloonTipTitle = $title
$notification.BalloonTipText = $body
$notification.Visible = $true
$notification.ShowBalloonTip(5000)
Start-Sleep -Milliseconds 5500
$notification.Dispose()
`, title64, body64)
}

func encodePowerShell(script string) string {
	encoded := utf16.Encode([]rune(script))
	raw := make([]byte, len(encoded)*2)
	for index, value := range encoded {
		raw[index*2] = byte(value)
		raw[index*2+1] = byte(value >> 8)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
