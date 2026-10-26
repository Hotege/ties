
#if !defined(_TIES_ADDONS_H_)
#define _TIES_ADDONS_H_

#include <stddef.h>

#ifdef __cplusplus
extern "C"
{
#endif

typedef struct AddonsHost {
    void (*SetStatus)(unsigned long long handle, int code);
    void (*SetHeader)(unsigned long long handle, const char* key, const char* value);
    int  (*Write)(unsigned long long handle, const char* data, size_t len);

    char* (*Query)(unsigned long long handle, const char* key);
    char* (*Param)(unsigned long long handle, const char* name);
    char* (*Form)(unsigned long long handle, const char* key);
    void* (*Body)(unsigned long long handle, size_t* len);
    void  (*Free)(void* p);
} AddonsHost;

void* AddonsLoad(const char* path);
size_t AddonsCount(void* ptr);
const char* AddonsGetName(void* ptr, size_t n);
size_t AddonsRouteCount(void* ptr, size_t n);
const char* AddonsGetMethod(void* ptr, size_t i, size_t j);
const char* AddonsGetRoute(void* ptr, size_t i, size_t j);
void AddonsHandle(unsigned long long handle, void* ptr, size_t i, size_t j, const AddonsHost* host);
void AddonsFree(void* ptr);

#ifdef __cplusplus
}
#endif

#endif
