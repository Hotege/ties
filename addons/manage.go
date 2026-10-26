package addons

/*
#cgo CFLAGS: -I .
#cgo LDFLAGS: -L ../libs -laddons

#include <stdlib.h>
#include "addons.h"

extern void GoCtxSetStatus(unsigned long long h, int code);
extern void GoCtxSetHeader(unsigned long long h, char* key, char* value);
extern int  GoCtxWrite(unsigned long long h, void* data, size_t n);

extern char* GoCtxQuery(unsigned long long h, char* key);
extern char* GoCtxParam(unsigned long long h, char* name);
extern char* GoCtxForm(unsigned long long h, char* key);
extern void* GoCtxBody(unsigned long long h, size_t* outLen);
extern void  GoCtxFree(void* p);
*/
import "C"
import (
    "runtime/cgo"
    "sync"
    "unsafe"

    "github.com/kataras/iris/v12"
)

var (
    mu sync.RWMutex
    data unsafe.Pointer
    host *C.AddonsHost
)

func ctxOf(h C.ulonglong) iris.Context {
    return cgo.Handle(uintptr(h)).Value().(iris.Context)
}

//export GoCtxSetStatus
func GoCtxSetStatus(h C.ulonglong, code C.int) {
    ctxOf(h).StatusCode(int(code))
}

//export GoCtxSetHeader
func GoCtxSetHeader(h C.ulonglong, key, value *C.char) {
    ctxOf(h).Header(C.GoString(key), C.GoString(value))
}

//export GoCtxWrite
func GoCtxWrite(h C.ulonglong, p unsafe.Pointer, n C.size_t) C.int {
    buf := C.GoBytes(p, C.int(n))
    written, err := ctxOf(h).Write(buf)
    if err != nil {
        return -1
    }
    return C.int(written)
}

//export GoCtxQuery
func GoCtxQuery(h C.ulonglong, key *C.char) *C.char {
    return C.CString(ctxOf(h).URLParam(C.GoString(key)))
}

//export GoCtxParam
func GoCtxParam(h C.ulonglong, name *C.char) *C.char {
    return C.CString(ctxOf(h).Params().Get(C.GoString(name)))
}

//export GoCtxForm
func GoCtxForm(h C.ulonglong, key *C.char) *C.char {
    return C.CString(ctxOf(h).PostValue(C.GoString(key)))
}

//export GoCtxBody
func GoCtxBody(h C.ulonglong, outLen *C.size_t) unsafe.Pointer {
    body, err := ctxOf(h).GetBody()
    if err != nil {
        *outLen = 0
        return nil
    }
    *outLen = C.size_t(len(body))
    return C.CBytes(body)
}

//export GoCtxFree
func GoCtxFree(p unsafe.Pointer) {
    C.free(p)
}

func Load(app *iris.Application, libs string) {
    mu.Lock()
    defer mu.Unlock()
    freeLocked()

    h := (*C.AddonsHost)(C.calloc(1, C.size_t(unsafe.Sizeof(C.AddonsHost{}))))
    h.SetStatus = (*[0]byte)(C.GoCtxSetStatus)
    h.SetHeader = (*[0]byte)(C.GoCtxSetHeader)
    h.Write = (*[0]byte)(C.GoCtxWrite)
    h.Query = (*[0]byte)(C.GoCtxQuery)
    h.Param = (*[0]byte)(C.GoCtxParam)
    h.Form = (*[0]byte)(C.GoCtxForm)
    h.Body = (*[0]byte)(C.GoCtxBody)
    h.Free = (*[0]byte)(C.GoCtxFree)

    cLibs := C.CString(libs)
    defer C.free(unsafe.Pointer(cLibs))

    d := C.AddonsLoad(cLibs)
    if d == nil {
        C.free(unsafe.Pointer(h))
        app.Logger().Error("addons: AddonsLoad returned NULL")
        return
    }

    data, host = d, h
    extract(app, d, h)
}

func extract(app *iris.Application, d unsafe.Pointer, h *C.AddonsHost) {
    count := int(C.AddonsCount(d))
    for n := 0; n < count; n++ {
        add(app, d, h, n)
    }
}

func add(app *iris.Application, d unsafe.Pointer, h *C.AddonsHost, n int) {
    routeCount := C.AddonsRouteCount(data, C.size_t(n))
    for i := range routeCount {
        cMethod := C.AddonsGetMethod(data, C.size_t(n), C.size_t(i))
        cRoute := C.AddonsGetRoute(data, C.size_t(n), C.size_t(i))
        if cMethod == nil || cRoute == nil {
            continue
        }
        method, route := C.GoString(cMethod), C.GoString(cRoute)
        app.Logger().Infof("Register: %s %s", method, route)
        app.Handle(method, route, func(ctx iris.Context) {
            mu.RLock()
            defer mu.RUnlock()
            if data != d {
                ctx.StatusCode(503)
                return
            }
            hd := cgo.NewHandle(ctx)
            defer hd.Delete()
            C.AddonsHandle(C.ulonglong(hd), d, C.size_t(n), C.size_t(i), h)
        })
    }
}

func freeLocked() {
    if data != nil {
        C.AddonsFree(data)
        data = nil
    }
    if host != nil {
        C.free(unsafe.Pointer(host))
        host = nil
    }
}

func Free() {
    mu.Lock()
    defer mu.Unlock()
    freeLocked()
}
