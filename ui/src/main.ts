import { EditorView, basicSetup } from "codemirror";
import { markdown } from "@codemirror/lang-markdown";
import { oneDark } from "@codemirror/theme-one-dark";
import { Compartment } from "@codemirror/state";

interface RenderOptions {
  pageSize: string;
  marginMm: number;
  baseFontPt: number;
  theme: string;
}

const defaultMarkdown = `# TypePDF — Markdown to PDF Converter

A **zero-server** Markdown to PDF converter running entirely in your browser via **Go + WebAssembly**. Every keystroke renders a live preview on the right.

No file uploads. No logins. No network requests after the initial page load. Your content never leaves your machine.

---

## How to Use

| Step | Action | Result |
|------|--------|--------|
| 1 | Type or paste Markdown in the left panel | PDF preview updates automatically |
| 2 | Adjust **Page Size**, **Margins**, or **Font Size** in the top bar | Preview re-renders instantly |
| 3 | Switch **Theme** to \`Draft\` for a cream paper background | Easier on the eyes for long editing |
| 4 | Click **Download PDF** when satisfied | Browser saves the PDF — no server involved |

> **Tip:** Use the **Draft** theme while editing, then switch to **Light** before downloading.

---

## Text Formatting

**Bold text**, *italic text*, ***bold and italic***, ~~strikethrough~~, \`inline code\`, and [hyperlinks](https://go.dev).

Automatic typographic replacements: (c) (C) (r) (R) (tm) (TM) (p) (P) +- ... -- --- "smart quotes" and 'single quotes'.

Bare URLs like https://github.com/yuin/goldmark are auto-linked.

---

## Lists

### Unordered
- **Bold item** with normal text following
- *Italic item* with \`inline code\` following
  - Nested with ~~strikethrough~~
  - Another nested item
    - Deeply nested (3 levels)

### Ordered
1. First item — note the numbered alignment
1. Second item with some wrapping text content
   1. Nested ordered A
   1. Nested ordered B
1. Third item

### Task List
- [x] Research Go WASM PDF engines
- [x] Build core rendering pipeline
- [ ] Add syntax highlighting to code blocks
- [ ] Ship v1.0

---

## Blockquotes

> **Note:** Blockquotes support *formatting*, \`code\`, and even nested quotes.
>
> > Nested blockquote with the accent bar on the left.
> >
> > > Triple nesting still renders correctly.

---

## Code Blocks

\`\`\`go
package main

import "fmt"

func main() {
    fmt.Println("Hello, TypePDF!")
    // This runs in WASM — no server needed
}
\`\`\`

    Indented code blocks work too, rendered with monospace font.

---

## Tables

| Syntax          | Supported | Since | Notes |
|-----------------|-----------|-------|-------|
| Headings H1-H6  | ✅ Yes    | v1.0  | Inter Bold, sized by level |
| Bold / Italic   | ✅ Yes    | v1.0  | Toggled mid-run |
| Strikethrough   | ✅ Yes    | v1.0  | Line at 50% cap height |
| Inline Code     | ✅ Yes    | v1.0  | Monospace, tinted bg |
| Links           | ✅ Yes    | v1.0  | Underlined, URL annotation |
| Images          | ✅ Yes    | v1.0  | Local files + remote placeholders |
| Code Blocks     | ✅ Yes    | v1.0  | Fenced + indented, paginated |
| Unordered Lists | ✅ Yes    | v1.0  | Nested up to 3 levels |
| Ordered Lists   | ✅ Yes    | v1.0  | Number alignment, nested |
| Task Lists      | ✅ Yes    | v1.1  | \`[x]\` and \`[ ]\` checkboxes |
| Blockquotes     | ✅ Yes    | v1.0  | Nested, with formatting |
| Thematic Break  | ✅ Yes    | v1.0  | Full-width line |
| Tables          | ✅ Yes    | v1.0  | Header accent, alternating rows |
| Typographic Repl| ✅ Yes    | v1.2  | Smart quotes, ellipsis, dashes |
| Autolink        | ✅ Yes    | v1.2  | Bare URLs become clickable links |

---

## Image

![Placeholder Image](https://placehold.co/600x400/EEE/31343C.png)

*Remote images show a placeholder if the URL is unreachable.*

---

## Why TypePDF?

- **Privacy first** — zero network requests after load
- **Offline capable** — works fully after first visit
- **No dependencies** — no Node.js, no Python, no Docker
- **Fast** — Go compiled to WASM renders at ~50ms for typical docs
- **Open source** — MIT licensed

---

> Built with [goldmark](https://github.com/yuin/goldmark) + [gopdf](https://github.com/signintech/gopdf) + [Go WASM](https://go.dev). Fonts: [Inter](https://rsms.me/inter/) under OFL 1.1.
`;

const MAX_MD_LENGTH = 50000;

let currentOptions: RenderOptions = {
  pageSize: "a4",
  marginMm: 25,
  baseFontPt: 11,
  theme: "light",
};

let worker: Worker | null = null;
let wasmReady = false;
let renderTimer: ReturnType<typeof setTimeout> | null = null;
let currentPdfUrl: string | null = null;

const editorEl = document.getElementById("editor")!;
const previewEl = document.getElementById("preview") as HTMLIFrameElement;
const loadingEl = document.getElementById("preview-loading")!;
const msgEl = document.getElementById("preview-msg")!;
const errorBarEl = document.getElementById("error-bar")!;
const downloadBtn = document.getElementById("btn-download") as HTMLButtonElement;

const pageSizeEl = document.getElementById("ctrl-page-size") as HTMLSelectElement;
const marginsEl = document.getElementById("ctrl-margins") as HTMLSelectElement;
const fontSizeEl = document.getElementById("ctrl-font-size") as HTMLSelectElement;
const themeEl = document.getElementById("ctrl-theme") as HTMLSelectElement;
const uiThemeEl = document.getElementById("ctrl-ui-theme") as HTMLSelectElement;

function showError(msg: string) {
  errorBarEl.textContent = msg;
  errorBarEl.style.display = "block";
}

function clearError() {
  errorBarEl.style.display = "none";
}

function showMsg(html: string) {
  msgEl.innerHTML = html;
  msgEl.style.display = "block";
}

function clearMsg() {
  msgEl.style.display = "none";
}

// --- Editor ---
const themeConfig = new Compartment();
const editor = new EditorView({
  doc: defaultMarkdown,
  extensions: [
    basicSetup,
    markdown(),
    themeConfig.of([]),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        scheduleRender();
      }
    }),
  ],
  parent: editorEl,
});

// --- Worker ---
function initWorker() {
  worker = new Worker("worker.js");

  worker.onmessage = (e) => {
    const msg = e.data;
    switch (msg.type) {
      case "ready":
        wasmReady = true;
        loadingEl.style.display = "none";
        scheduleRender();
        break;
      case "result":
        clearError();
        showPdf(msg.bytes);
        break;
      case "error":
        showError("Render error: " + msg.message);
        break;
    }
  };
}

function requestRender(markdown: string, options: RenderOptions) {
  if (!worker || !wasmReady) return;
  worker.postMessage({ type: "render", markdown, options });
}

function scheduleRender() {
  if (!wasmReady) return;
  if (renderTimer) clearTimeout(renderTimer);
  renderTimer = setTimeout(() => {
    const md = editor.state.doc.toString();
    if (md.length > MAX_MD_LENGTH) {
      showError("Markdown exceeds " + MAX_MD_LENGTH.toLocaleString() + " character limit (" + md.length.toLocaleString() + " chars). Please shorten.");
      downloadBtn.disabled = true;
      return;
    }
    clearError();
    requestRender(md, currentOptions);
  }, 300);
}

// --- PDF Preview ---
function showPdf(bytes: Uint8Array) {
  clearMsg();
  if (currentPdfUrl) URL.revokeObjectURL(currentPdfUrl);
  const blob = new Blob([bytes as any], { type: "application/pdf" });
  currentPdfUrl = URL.createObjectURL(blob);
  previewEl.src = currentPdfUrl;
  downloadBtn.disabled = false;
}

// --- Download ---
downloadBtn.addEventListener("click", () => {
  if (!currentPdfUrl) return;
  const a = document.createElement("a");
  a.href = currentPdfUrl;
  a.download = "document.pdf";
  a.click();
});

// --- Controls ---
function readControls(): RenderOptions {
  return {
    pageSize: pageSizeEl.value,
    marginMm: parseInt(marginsEl.value),
    baseFontPt: parseInt(fontSizeEl.value),
    theme: themeEl.value,
  };
}

function onControlChange() {
  currentOptions = readControls();
  scheduleRender();
}

pageSizeEl.addEventListener("change", onControlChange);
marginsEl.addEventListener("change", onControlChange);
fontSizeEl.addEventListener("change", onControlChange);
themeEl.addEventListener("change", onControlChange);

// --- UI Theme ---
function applyUiTheme() {
  const val = uiThemeEl.value;
  const isDark = val === "dark" || (val === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
  
  if (isDark) {
    document.body.classList.add("dark-theme");
    editor.dispatch({ effects: themeConfig.reconfigure([oneDark]) });
  } else {
    document.body.classList.remove("dark-theme");
    editor.dispatch({ effects: themeConfig.reconfigure([]) });
  }
}

uiThemeEl.addEventListener("change", applyUiTheme);
window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", (e) => {
  if (uiThemeEl.value === "system") applyUiTheme();
});

applyUiTheme();

// --- Init ---
initWorker();
