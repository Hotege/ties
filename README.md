# ties

Write HTTP handlers in C++, compile them into shared libraries, drop them into a directory, and the server loads them and registers their routes automatically. Change a library file and the server hot-reloads.

`ties` has two parts:

- **Host (Go):** an HTTP server built on [Iris](https://github.com/kataras/iris). It listens on the port, parses requests and watches the filesystem for changes.
- **Plugins (C++):** each plugin is a shared library with the `.tie` extension. It declares the `method + path` pairs it handles, and when a request arrives it reads parameters and writes the response through callbacks provided by the host.

## Features

- Plugins are plain shared libraries: build them with any C/C++ toolchain. No need to link against the host, only one header to include.
- A single plugin can register multiple routes.
- Hot reload: creating, modifying, deleting or renaming a `.tie` file in the `libs` directory restarts the HTTP server and reloads all plugins after a 300 ms debounce.
- Request handling holds a read lock. Reloading waits for in-flight requests to finish before releasing the old plugins, and requests that land on a replaced plugin get `503`, so a freed plugin is never touched.
- A multi-stage `Dockerfile` is included; the runtime image contains only runtime dependencies.

## How it works

```
request   →  Iris router  →  cgo  →  libaddons.so  →  plugin's AddonHandle()
response  ←  host callbacks (SetStatus / SetHeader / Write / Query …)  ←  plugin
```

1. On startup `ties` calls `AddonsLoad` in `libaddons.so`, which scans every `.tie` file in the `-libs` directory (non-recursive).
2. For each library that exports both `AddonName` and `AddonCreate`, a plugin instance is created. If two libraries report the same `AddonName`, only the first one loaded is kept.
3. Each plugin's route table is read and registered with Iris via `app.Handle(method, route, ...)`.
4. When a request arrives, the host wraps the request context in a handle and calls the plugin's `AddonHandle`. The plugin reads the request and writes the response through the function pointers in `AddonsHost`.

## Project layout

```
ties/
├── main.go               # Entry point: HTTP server, .tie file watching, hot reload
├── addons/
│   ├── manage.go         # cgo bridge: load/free plugins, register routes, expose host callbacks
│   ├── addons.h          # C interface shared by host and plugins (incl. the AddonsHost callback table)
│   └── defines.h         # Symbol export macro ADDONS_API
├── cxx/addons.cpp        # libaddons.so: scans and loads .tie files (via boost::dll)
├── subjects/hello/       # Example plugin
├── projects/
│   ├── build.sh          # Builds libaddons.so, the example plugin and the ties binary
│   └── run.sh            # Sets LD_LIBRARY_PATH and starts the server
└── Dockerfile
```

Build outputs (all ignored by `.gitignore`):

| Path | Description |
| --- | --- |
| `libs/libaddons.so` | The plugin loader. `ties` depends on it at build time and at run time |
| `libs/*.tie` | Plugins, loaded from here at run time |
| `outputs/ties` | The host executable |

## Requirements

- Linux (the build and start scripts are bash and rely on `.so` files and `LD_LIBRARY_PATH`)
- Go 1.27.3 or newer (see `go.mod`) with cgo enabled
- `g++` with C++17 support (the code uses `std::filesystem`)
- Boost: `filesystem`, plus the header-only `dll` library

Ubuntu / Debian:

```bash
sudo apt-get install g++ libboost-filesystem-dev
```

## Quick start

```bash
git clone https://github.com/Hotege/ties.git
cd ties

# libs/ and outputs/ are not in the repository; create them first
mkdir -p libs outputs

./projects/build.sh   # builds libs/libaddons.so, libs/hello.tie and outputs/ties, in that order
./projects/run.sh     # listens on :8080 by default
```

In another terminal:

```bash
curl 'http://localhost:8080/hello'
# Hello World!

curl 'http://localhost:8080/hello?name=Ties'
# Hello Ties!
```

> The build order matters: `ties` links against `libs/libaddons.so` through cgo, so that library must be built first.

## Command-line flags

| Flag | Default | Description |
| --- | --- | --- |
| `-libs` | `.` | Directory holding `libaddons.so` and the `.tie` plugins; this is also the directory that is watched |

`run.sh` points `LD_LIBRARY_PATH` at the `libs` directory under the current directory and starts `outputs/ties` with that directory's absolute path as `-libs`. Any extra arguments are passed through unchanged, so run it from the repository root.

The listening port is currently fixed at `:8080` (hard-coded in `main.go`).

## Writing a plugin

See [`subjects/hello/hello.cpp`](subjects/hello/hello.cpp) for a complete example.

### Functions to export

Every function is exported with `extern "C"` and marked with `ADDONS_API` (from `defines.h`).

| Function | Required | Description |
| --- | --- | --- |
| `const char* AddonName()` | Yes | Plugin name; must be unique. A later library with the same name is ignored |
| `void* AddonCreate()` | Yes | Creates the plugin instance. The return value is passed to the other functions as `ptr`; it may be `nullptr` |
| `size_t AddonRouteCount(void* ptr)` | No | Number of routes. If missing it counts as 0, i.e. no routes are registered |
| `const char* AddonMethod(void* ptr, size_t n)` | No | HTTP method of route `n`, e.g. `"GET"` |
| `const char* AddonRoute(void* ptr, size_t n)` | No | Path of route `n`, in Iris route syntax, e.g. `"/users/{id}"` |
| `void AddonHandle(unsigned long long handle, void* ptr, size_t n, const AddonsHost* host)` | No | Handles a request for route `n` |
| `void AddonDestroy(void* ptr)` | No | Called on unload; use it to free the instance created by `AddonCreate` |

`n` ranges over `0 .. AddonRouteCount-1`, and the `n` passed to `AddonMethod`, `AddonRoute` and `AddonHandle` refers to the same route.

### Host callbacks (`AddonsHost`)

The `host` argument of `AddonHandle` provides the following functions. The first parameter of each is the request `handle`:

| Callback | Description |
| --- | --- |
| `SetStatus(handle, code)` | Set the status code |
| `SetHeader(handle, key, value)` | Set a response header |
| `Write(handle, data, len)` | Write the response body; returns the number of bytes written, or `-1` on failure |
| `Query(handle, key)` | Get a URL query parameter |
| `Param(handle, name)` | Get a path parameter |
| `Form(handle, key)` | Get a form field |
| `Body(handle, &len)` | Get the raw request body; its length is returned through `len`. Returns `NULL` with length 0 if reading fails |
| `Free(p)` | Free memory returned by the functions above |

Notes:

- Memory returned by `Query`, `Param`, `Form` and `Body` is allocated by the host and **must be released with `host->Free`**, not with `free` or `delete`.
- Call `SetStatus` / `SetHeader` before `Write`.

### Minimal example

```cpp
#include <cstddef>
#include <string>
#include <defines.h>
#include <addons.h>

extern "C" {

ADDONS_API const char* AddonName()    { return "ping"; }
ADDONS_API void*       AddonCreate()  { return nullptr; }
ADDONS_API void        AddonDestroy(void*) {}

ADDONS_API size_t AddonRouteCount(void*) { return 1; }

ADDONS_API const char* AddonMethod(void*, size_t n) { return n == 0 ? "GET" : nullptr; }
ADDONS_API const char* AddonRoute(void*, size_t n)  { return n == 0 ? "/ping" : nullptr; }

ADDONS_API void AddonHandle(unsigned long long handle, void*, size_t n, const AddonsHost* host) {
    if (n != 0) return;
    const std::string out = "pong\n";
    host->SetStatus(handle, 200);
    host->SetHeader(handle, "Content-Type", "text/plain; charset=utf-8");
    host->Write(handle, out.data(), out.size());
}

}
```

### Building and deploying

A plugin does not link against `libaddons`; it only needs the headers in this repository:

```bash
g++ subjects/ping/ping.cpp -shared -fPIC -s -o libs/ping.tie -I addons
```

Put the result in the directory given by `-libs` (the extension must be `.tie`). If the server is running, a hot reload is triggered automatically.

## Hot reload

- Only the `-libs` directory itself is watched (not recursively), and only write, create, remove and rename events on `.tie` files count. Changes to other files are ignored.
- After an event, the server waits for 300 ms without further events, then reloads: it gracefully shuts down the current HTTP server (waiting up to 5 seconds), frees all plugins, then rescans and starts again.
- `libaddons.so` and `outputs/ties` themselves are not covered by hot reload; updating them requires restarting the process.

## Docker

The `Dockerfile` has two stages:

1. **builder:** based on `ubuntu:26.04`; installs `g++` and Boost, downloads Go, and runs `projects/build.sh`.
2. **runtime:** based on `ubuntu:26.04`; contains only `libboost-filesystem` and `libstdc++6`, and copies `libs/`, `outputs/` and `run.sh` from the builder.

```bash
mkdir -p libs outputs          # same as above: the build script expects both directories to exist
docker build -t ties .
docker run --rm -p 8080:8080 ties
```

The image sets the Go module proxy to `https://goproxy.cn,direct` and the checksum database to `sum.golang.google.cn`. If you are not building from mainland China, change these back to the defaults in the `Dockerfile`.
