//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"
)

func main() {
	c := make(chan struct{}, 0)
	js.Global().Set("compileMarkdownToPdf", js.FuncOf(jsCompileMarkdownToPdf))
	<-c
}

func jsCompileMarkdownToPdf(this js.Value, args []js.Value) interface{} {
	promiseConstructor := js.Global().Get("Promise")
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 1 {
				reject.Invoke(js.Global().Get("Error").New("missing markdown argument"))
				return
			}

			markdown := args[0].String()

			config := DefaultConfig()
			if len(args) >= 2 {
				optionsJSON := args[1].String()
				if optionsJSON != "" {
					var parsed Config
					if err := json.Unmarshal([]byte(optionsJSON), &parsed); err == nil {
						if parsed.PageSize != "" {
							config.PageSize = parsed.PageSize
						}
						if parsed.MarginMm != 0 {
							config.MarginMm = parsed.MarginMm
						}
						if parsed.BaseFontPt != 0 {
							config.BaseFontPt = parsed.BaseFontPt
						}
						if parsed.Theme != "" {
							config.Theme = parsed.Theme
						}
						if parsed.BaseDir != "" {
							config.BaseDir = parsed.BaseDir
						}
					}
				}
			}

			pdfBytes, err := renderWithBaseDir([]byte(markdown), config, config.BaseDir)
			if err != nil {
				reject.Invoke(js.Global().Get("Error").New(err.Error()))
				return
			}

			arr := js.Global().Get("Uint8Array").New(len(pdfBytes))
			js.CopyBytesToJS(arr, pdfBytes)
			resolve.Invoke(arr)
		}()
		return nil
	})
	return promiseConstructor.New(handler)
}
