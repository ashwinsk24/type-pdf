importScripts("wasm_exec.js");

const go = new Go();

let ready = false;

WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject)
  .then((r) => {
    go.run(r.instance);
    ready = true;
    postMessage({ type: "ready" });
  })
  .catch((err) => {
    console.error("WASM init error:", err);
    postMessage({ type: "error", message: err.message });
  });

onmessage = (e) => {
  if (e.data.type === "render") {
    if (!ready) {
      postMessage({ type: "error", message: "WASM not ready" });
      return;
    }
    try {
      const { markdown, options } = e.data;
      globalThis.compileMarkdownToPdf(markdown, JSON.stringify(options))
        .then(bytes => {
          const bytesCopy = new Uint8Array(bytes);
          postMessage({ type: "result", bytes: bytesCopy }, [bytesCopy.buffer]);
        })
        .catch(err => {
          postMessage({ type: "error", message: err.message || err.toString() });
        });
    } catch (err) {
      postMessage({ type: "error", message: err.message });
    }
  }
};
