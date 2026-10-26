
#if !defined(_TIES_ADDONS_DEFINES_H_)
#define _TIES_ADDONS_DEFINES_H_

#if defined(_WIN32) || defined(_WIN64)
#ifdef STRATEGY_BUILD_LIB
#define ADDONS_API __declspec(dllexport)
#else
#define ADDONS_API __declspec(dllimport)
#endif
#else
#define ADDONS_API __attribute__((visibility("default")))
#endif

#endif
