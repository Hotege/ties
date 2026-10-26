
#include <addons.h>
#include <filesystem>
#include <map>
#include <string>
#include <tuple>

#include <boost/dll/import.hpp>

namespace fs = std::filesystem;
using library = boost::dll::shared_library;

struct Context {
    std::vector<std::string> names;
    std::map<std::string, std::tuple<std::shared_ptr<library>, void*>> libs;
};

void* AddonsLoad(const char* path) {
    Context* ctx = new Context;
    try {
        fs::path dir = path;
        for (const auto& entry : fs::directory_iterator(dir)) {
            if (!entry.is_regular_file()) {
                continue;
            }
            if (entry.path().extension() != ".tie") {
                continue;
            }
            std::shared_ptr<library> L = nullptr;
            try {
                L = std::make_shared<library>(entry.path().string(), boost::dll::load_mode::append_decorations);
            } catch (...) {
                L = nullptr;
            }
            if (!L) {
                continue;
            }
            if (!L->has("AddonName")) {
                continue;
            }
            auto name = L->get<const char*()>("AddonName");
            if (ctx->libs.count(name())) {
                continue;
            }
            if (!L->has("AddonCreate")) {
                continue;
            }
            auto create = L->get<void*()>("AddonCreate");
            ctx->names.push_back(name());
            ctx->libs[name()] = { L, create() };
        }
        return (void*)ctx;
    } catch (...) {
        delete ctx;
        return nullptr;
    }
}

size_t AddonsCount(void* ptr) {
    try {
        Context* ctx = (Context*)ptr;
        return ctx->names.size();
    } catch (...) {
        return 0;
    }
}

const char* AddonsGetName(void* ptr, size_t n) {
    try {
        Context* ctx = (Context*)ptr;
        return ctx->names[n].c_str();
    } catch (...) {
        return nullptr;
    }
}

size_t AddonsRouteCount(void* ptr, size_t n) {
    try {
        Context* ctx = (Context*)ptr;
        auto& data = ctx->libs.at(ctx->names.at(n));
        auto& L = std::get<0>(data);
        auto& P = std::get<1>(data);
        if (L->has("AddonRouteCount")) {
            auto func = L->get<size_t(void*)>("AddonRouteCount");
            return func(P);
        } else {
            return 0;
        }
    } catch (...) {
        return 0;
    }
}

const char* AddonsGetMethod(void* ptr, size_t i, size_t j) {
    try {
        Context* ctx = (Context*)ptr;
        auto& data = ctx->libs.at(ctx->names.at(i));
        auto& L = std::get<0>(data);
        auto& P = std::get<1>(data);
        if (L->has("AddonMethod")) {
            auto func = L->get<const char*(void*, size_t)>("AddonMethod");
            return func(P, j);
        } else {
            return nullptr;
        }
    } catch (...) {
        return nullptr;
    }
}

const char* AddonsGetRoute(void* ptr, size_t i, size_t j) {
    try {
        Context* ctx = (Context*)ptr;
        auto& data = ctx->libs.at(ctx->names.at(i));
        auto& L = std::get<0>(data);
        auto& P = std::get<1>(data);
        if (L->has("AddonRoute")) {
            auto func = L->get<const char*(void*, size_t)>("AddonRoute");
            return func(P, j);
        } else {
            return nullptr;
        }
    } catch (...) {
        return nullptr;
    }
}

void AddonsHandle(unsigned long long handle, void* ptr, size_t i, size_t j, const AddonsHost* host) {
    try {
        Context* ctx = (Context*)ptr;
        auto& data = ctx->libs.at(ctx->names.at(i));
        auto& L = std::get<0>(data);
        auto& P = std::get<1>(data);
        if (L->has("AddonHandle")) {
            auto func = L->get<void(unsigned long long, void*, size_t, const AddonsHost*)>("AddonHandle");
            func(handle, P, j, host);
        }
    } catch (...) {
        return;
    }
}

void AddonsFree(void* ptr) {
    try {
        Context* ctx = (Context*)ptr;
        for (auto& kv : ctx->libs) {
            auto& L = std::get<0>(kv.second);
            auto& addon = std::get<1>(kv.second);
            if (L->has("AddonDestroy")) {
                auto destroy = L->get<void(void*)>("AddonDestroy");
                destroy(addon);
                addon = nullptr;
            }
        }
        delete ctx;
    } catch (...) {
        return;
    }
}
