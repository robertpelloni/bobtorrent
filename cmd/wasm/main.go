//go:build js && wasm

package main

import (
	"encoding/hex"
	"syscall/js"
	"bobtorrent/pkg/storage"
)

func main() {
	c := make(chan struct{}, 0)
	println("Bobtorrent Storage WASM Initialized")
	
	js.Global().Set("bobEncrypt", js.FuncOf(encryptWrapper))
	js.Global().Set("bobDecrypt", js.FuncOf(decryptWrapper))
	js.Global().Set("bobEncodeErasure", js.FuncOf(encodeErasureWrapper))
	js.Global().Set("bobDecodeErasure", js.FuncOf(decodeErasureWrapper))

	<-c
}

func encryptWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 { return nil }
	data := make([]byte, args[0].Length())
	js.CopyBytesToGo(data, args[0])

	s, _ := storage.NewStorage("", 4, 2)
	blob, key, nonce, err := s.EncryptChunk(data)
	if err != nil { return js.ValueOf(err.Error()) }

	result := js.Global().Get("Object").New()
	dstBlob := js.Global().Get("Uint8Array").New(len(blob))
	js.CopyBytesToJS(dstBlob, blob)
	result.Set("blob", dstBlob)
	result.Set("key", hex.EncodeToString(key))
	result.Set("nonce", hex.EncodeToString(nonce))
	return result
}

func decryptWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 3 { return nil }
	blob := make([]byte, args[0].Length())
	js.CopyBytesToGo(blob, args[0])
	key, _ := hex.DecodeString(args[1].String())
	nonce, _ := hex.DecodeString(args[2].String())

	s, _ := storage.NewStorage("", 4, 2)
	plain, err := s.DecryptChunk(blob, key, nonce)
	if err != nil { return js.ValueOf(err.Error()) }

	dstPlain := js.Global().Get("Uint8Array").New(len(plain))
	js.CopyBytesToJS(dstPlain, plain)
	return dstPlain
}

func encodeErasureWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 { return nil }
	data := make([]byte, args[0].Length())
	js.CopyBytesToGo(data, args[0])

	coder, _ := storage.NewErasureCoder(4, 2)
	shards, err := coder.Encode(data)
	if err != nil { return js.ValueOf(err.Error()) }

	result := js.Global().Get("Array").New(len(shards))
	for i, shard := range shards {
		dstShard := js.Global().Get("Uint8Array").New(len(shard))
		js.CopyBytesToJS(dstShard, shard)
		result.SetIndex(i, dstShard)
	}
	return result
}

func decodeErasureWrapper(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 { return nil }
	jsShards := args[0]
	shards := make([][]byte, jsShards.Length())
	for i := 0; i < len(shards); i++ {
		val := jsShards.Index(i)
		if val.Type() != js.TypeUndefined && val.Type() != js.TypeNull {
			shards[i] = make([]byte, val.Length())
			js.CopyBytesToGo(shards[i], val)
		}
	}

	coder, _ := storage.NewErasureCoder(4, 2)
	err := coder.Reconstruct(shards)
	if err != nil { return js.ValueOf(err.Error()) }

	blob, err := coder.Join(shards, storage.FixedBlobSize)
	if err != nil { return js.ValueOf(err.Error()) }

	dstBlob := js.Global().Get("Uint8Array").New(len(blob))
	js.CopyBytesToJS(dstBlob, blob)
	return dstBlob
}
