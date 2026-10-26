
#include <cstddef>
#include <string>
#include <defines.h>
#include <addons.h>

extern "C"
{

ADDONS_API const char* AddonName() {
    return "hello";
}

ADDONS_API void* AddonCreate() {
    return nullptr;
}

ADDONS_API size_t AddonRouteCount(void* ptr) {
    return 1;
}

ADDONS_API const char* AddonMethod(void* ptr, size_t n) {
    if (n == 0) {
        return "GET";
    }
    return nullptr;
}

ADDONS_API const char* AddonRoute(void* ptr, size_t n) {
    if (n == 0) {
        return "/hello";
    }
    return nullptr;
}

static std::string take(const AddonsHost* host, char* p) {
    if (!p) {
        return {};
    }
    std::string s(p);
    host->Free(p);
    return s;
}

ADDONS_API void AddonHandle(unsigned long long handle, void* ptr, size_t n, const AddonsHost* host) {
    if (n == 0) {
        std::string out;
        std::string name = take(host, host->Query(handle, "name"));
        out = "Hello " + (name.empty() ? "World" : name) + "!\n";
        host->SetStatus(handle, 200);
        host->SetHeader(handle, "Content-Type", "text/plain; charset=utf-8");
        host->Write(handle, out.data(), out.size());
    }
}

ADDONS_API void AddonDestroy(void* ptr) {
}

}
