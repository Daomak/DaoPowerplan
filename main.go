package main

import (
	"bytes"
	"embed"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"golang.org/x/sys/windows"
	"golang.org/x/text/encoding/simplifiedchinese"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 创建命名互斥体，防止多开
	mutexName, _ := windows.UTF16PtrFromString("Global\\DaoPowerplan_SingleInstance")
	mutex, err := windows.CreateMutex(nil, false, mutexName)
	if err != nil {
		// 互斥体已存在或创建失败，直接退出
		os.Exit(0)
	}
	defer windows.CloseHandle(mutex)

	app := NewApp()
	err = wails.Run(&options.App{
		Title:  "DaoPowerplan",
		Width:  960,
		Height: 720,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}

func hideWindow() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}

func runPowercfg(args ...string) (string, error) {
	cmd := exec.Command("powercfg", args...)
	cmd.SysProcAttr = hideWindow()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	// 转换GBK到UTF-8
	output := stdout.String()
	if stderr.Len() > 0 {
		output += stderr.String()
	}
	decoded, decodeErr := gbkToUtf8(output)
	if decodeErr == nil {
		output = decoded
	}
	return strings.TrimSpace(output), err
}

func gbkToUtf8(s string) (string, error) {
	reader := simplifiedchinese.GBK.NewDecoder().Reader(strings.NewReader(s))
	result, err := io.ReadAll(reader)
	if err != nil {
		return s, err
	}
	return string(result), nil
}
