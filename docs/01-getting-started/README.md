# Getting started

ember is a Go game engine for online, open-world games. A game is plain Go: entities are structs, systems are small objects, and the same World runs in a window or headless on a server.

This page gets an example running, then shows how to ship a game and run it in the browser.

## What you need

- **Go 1.27.0** or later.
- **A C compiler** for the windowed build, which reaches GLFW and WebGPU through cgo. On macOS: `xcode-select --install`.
- **A graphics card WebGPU can draw with:**
  - macOS: Metal.
  - Windows: Direct3D 12, Vulkan, or else OpenGL.
  - Linux: Vulkan, or OpenGL ES where there's no Vulkan driver. Either way, install the Vulkan loader (`libvulkan1`).
- **The WebGPU library**, wgpu-native, which ember opens when it starts. `task wgpu` fetches it for `go run`; a built game ships it beside its executable.

A server needs neither the compiler nor the library: the packages it uses are pure Go and build with `CGO_ENABLED=0`.

## Run the examples

Clone the repository, fetch the WebGPU library once, and run an example from the repository's folder:

```sh
git clone https://github.com/lantervale/ember
cd ember
task wgpu
go run ./examples/01-hello-cube
```

`task wgpu` puts the library in the source tree, where `go run` finds it.

Each page shows its example's source as it is in `examples`, included when the site is built, so the code you read is the code you run.

Start with [Hello cube](01-hello-cube.md), then [The World](../02-world/README.md).

## Ship a game

`task package` builds a game into `dist/` with its WebGPU library beside it:

```sh
task package PKG=./examples/19-sprites NAME=sprites
```

Set `GOOS` and `GOARCH` to build for another system.

| System | What `dist/` holds | To know |
|---|---|---|
| macOS | `sprites-darwin-<arch>/sprites.app`: the game in `Contents/MacOS`, `libwgpu_native.dylib` in `Contents/Frameworks`, and its `Info.plist` | The arch is in the folder name, so arm64 and amd64 builds don't overwrite each other. The app is signed ad hoc: it runs on the machine that built it, and a copy downloaded elsewhere shows Gatekeeper's warning until it is signed with a Developer ID and notarized. |
| Windows | `sprites-windows-<arch>/`: `sprites.exe`, a windowed build with no console, beside `wgpu_native.dll` | It builds from macOS too, with the Windows C compiler named in `CC`, or else mingw-w64 for amd64 (`brew install mingw-w64`) and zig for arm64 (`brew install zig`), since mingw-w64 has no arm64 compiler. |
| Linux | `sprites-linux-<arch>/`: the game beside `libwgpu_native.so` | It builds on Linux, with GLFW's X11 and Wayland headers and `libvulkan-dev`. The game links the Vulkan loader rather than OpenGL, so players need `libvulkan1`, even on a card that draws through OpenGL ES; nearly every desktop distribution has it. |

### Where a game finds the library

A game looks for wgpu-native in this order:

1. the path in `EMBER_WGPU_LIB`;
2. beside its executable;
3. in a macOS app's `Contents/Frameworks`;
4. in the source tree, where `task wgpu` put it (not for a build made with `-trimpath`).

Run a game with `EMBER_STATS=1` to see what it drew with: when it quits, it prints a line such as `graphics: WebGPU (wgpu-native 29.0.0.0, Metal)` and the graphics card.

## Run in the browser

The guide's windowed examples also run in the page, built for WebAssembly and drawn with WebGPU. When a page can't run them, it shows the screenshot and says why.

The browser needs two things:

- **WebGPU.** Recent desktop Chrome, Edge, Safari and Firefox have it.
- **A secure page.** Browsers give WebGPU only to pages over https or from localhost. A guide served over http from a network address, such as a phone reaching `http://192.168.…`, has none.

### Turn WebGPU on

| Browser | WebGPU |
|---|---|
| Chrome, Edge | On by default on Windows, macOS, ChromeOS and Android. On Linux, turn on `chrome://flags/#enable-unsafe-webgpu` and restart. |
| Safari | On by default from Safari 26. Before that: Settings › Advanced › "Show features for web developers", then Develop › Feature Flags › WebGPU. |
| Firefox | On by default on Windows and macOS in recent versions. Elsewhere, set `dom.webgpu.enabled` to true in `about:config` and restart. |

WebGPU is also off when the browser blocks the graphics driver, or when hardware acceleration is off (Chrome and Edge: Settings › System). Update the driver or turn acceleration back on. [webgpureport.org](https://webgpureport.org) shows what your browser offers.

### Build an example for the browser

`task web PKG=./examples/19-sprites` builds one into `dist/web`, with its page. Serve that folder from localhost and open it:

```sh
python3 -m http.server -d dist/web
```
