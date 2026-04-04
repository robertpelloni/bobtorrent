/*
 * Bobtorrent Storage WASM Loader
 * --------------------------------------------------------------------------
 * This browser-side helper bootstraps the Go WebAssembly runtime and exposes
 * a clean Promise-based API over the low-level globals exported by
 * cmd/wasm/main.go:
 *   - bobEncrypt(Uint8Array)
 *   - bobDecrypt(Uint8Array, keyHex, nonceHex)
 *   - bobEncodeErasure(Uint8Array)
 *   - bobDecodeErasure(Array<Uint8Array|null>)
 *
 * Why this file exists:
 *   1. Frontend code should not have to know about Go's `wasm_exec.js` API.
 *   2. The bobcoin React app can import or copy this loader directly.
 *   3. The loader normalizes outputs into predictable JS objects.
 *
 * Usage:
 *   import { createBobtorrentStorageClient } from './storage-wasm-loader.js';
 *
 *   const client = await createBobtorrentStorageClient({
 *     wasmExecUrl: '/wasm_exec.js',
 *     wasmBinaryUrl: '/storage.wasm'
 *   });
 *
 *   const encrypted = await client.encrypt(fileBytes);
 *   const shards = await client.encodeErasure(encrypted.blob);
 */

function assertBrowser() {
  if (typeof window === 'undefined') {
    throw new Error('storage-wasm-loader.js must run in a browser environment');
  }
}

function ensureUint8Array(value, name) {
  if (!(value instanceof Uint8Array)) {
    throw new TypeError(`${name} must be a Uint8Array`);
  }
}

async function loadScript(src) {
  await new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[data-bobtorrent-wasm="${src}"]`);
    if (existing) {
      existing.addEventListener('load', resolve, { once: true });
      existing.addEventListener('error', reject, { once: true });
      if (existing.dataset.loaded === 'true') resolve();
      return;
    }

    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    script.dataset.bobtorrentWasm = src;
    script.onload = () => {
      script.dataset.loaded = 'true';
      resolve();
    };
    script.onerror = () => reject(new Error(`Failed to load script: ${src}`));
    document.head.appendChild(script);
  });
}

async function loadGoRuntime(wasmExecUrl) {
  if (typeof window.Go !== 'undefined') {
    return window.Go;
  }
  await loadScript(wasmExecUrl);
  if (typeof window.Go === 'undefined') {
    throw new Error('Go runtime did not initialize after loading wasm_exec.js');
  }
  return window.Go;
}

let runtimePromise = null;

/**
 * Initializes the Go runtime and executes storage.wasm exactly once.
 */
export async function initBobtorrentStorageWasm(options = {}) {
  assertBrowser();

  const {
    wasmExecUrl = '/wasm_exec.js',
    wasmBinaryUrl = '/storage.wasm',
  } = options;

  if (runtimePromise) {
    return runtimePromise;
  }

  runtimePromise = (async () => {
    const GoRuntime = await loadGoRuntime(wasmExecUrl);
    const go = new GoRuntime();

    const response = await fetch(wasmBinaryUrl);
    if (!response.ok) {
      throw new Error(`Failed to fetch WASM binary: ${response.status} ${response.statusText}`);
    }

    const bytes = await response.arrayBuffer();
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);

    // Run the Go WASM module. This call intentionally is not awaited because
    // the Go runtime blocks forever on a channel to keep the exports alive.
    go.run(instance);

    const requiredExports = ['bobEncrypt', 'bobDecrypt', 'bobEncodeErasure', 'bobDecodeErasure'];
    for (const fn of requiredExports) {
      if (typeof window[fn] !== 'function') {
        throw new Error(`Go WASM export missing: ${fn}`);
      }
    }

    return {
      encrypt: (input) => {
        ensureUint8Array(input, 'input');
        const result = window.bobEncrypt(input);
        if (typeof result === 'string') {
          throw new Error(result);
        }
        return {
          blob: result.blob,
          key: result.key,
          nonce: result.nonce,
        };
      },
      decrypt: (ciphertext, keyHex, nonceHex) => {
        ensureUint8Array(ciphertext, 'ciphertext');
        const result = window.bobDecrypt(ciphertext, keyHex, nonceHex);
        if (typeof result === 'string') {
          throw new Error(result);
        }
        return result;
      },
      encodeErasure: (input) => {
        ensureUint8Array(input, 'input');
        const shards = window.bobEncodeErasure(input);
        if (typeof shards === 'string') {
          throw new Error(shards);
        }
        return Array.from(shards);
      },
      decodeErasure: (shards) => {
        if (!Array.isArray(shards)) {
          throw new TypeError('shards must be an array');
        }
        const result = window.bobDecodeErasure(shards);
        if (typeof result === 'string') {
          throw new Error(result);
        }
        return result;
      },
    };
  })();

  return runtimePromise;
}

/**
 * Convenience factory mirroring the client style frontend code typically uses.
 */
export async function createBobtorrentStorageClient(options = {}) {
  return initBobtorrentStorageWasm(options);
}
