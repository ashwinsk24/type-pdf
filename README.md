# TypePDF (md2pdf)

A lightning-fast, offline-capable Markdown-to-PDF engine running entirely in your browser using WebAssembly.

## Overview
TypePDF compiles Markdown directly into PDF using a custom Go-based WebAssembly engine. By bypassing Node.js/Puppeteer dependencies and HTML-to-PDF conversion steps, it achieves extremely fast render times (~50ms) completely offline.

### Features
- **Offline Capable:** Full PDF compilation occurs locally in the browser.
- **Privacy First:** Zero network requests are required to generate your document.
- **WASM Engine:** Custom high-performance rendering engine written in Go and `gopdf`.
- **Live Preview:** Instant IDE-like side-by-side editing experience.
- **Rich Formatting:** Supports headings, blockquotes, tables, task lists, code blocks, and dynamic image loading.

## Tech Stack
- **Engine:** Go (`GOOS=js GOARCH=wasm`), `gopdf`, `goldmark`
- **UI:** TypeScript, Vite, CodeMirror 6

## Development Setup

### 1. Build the WASM Engine
You need Go 1.21+ installed on your machine.
```bash
cd engine
GOOS=js GOARCH=wasm go build -o main.wasm .
cp main.wasm ../ui/public/main.wasm
```

### 2. Run the Engine Tests (Optional)
Requires Node.js 20+.
```bash
cd engine
node test-harness.js
```

### 3. Run the UI Development Server
Requires Node.js and NPM.
```bash
cd ui
npm install
npm run dev
```
Navigate to `http://localhost:5173/` in your browser.

## CI/CD
A GitHub Actions workflow is included (`.github/workflows/ci.yml`) to automatically build the WASM engine, run the snapshot regression tests via the Node.js harness, and verify the UI builds successfully.

## License
MIT
