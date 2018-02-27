#ifndef HELPER_H
#define HELPER_H

#include <stdbool.h>
#include <libyang/libyang.h>
#include <libyang/tree_data.h>

typedef const struct ly_ctx const_ctx;

typedef void (*clb)(LY_LOG_LEVEL level, const char *msg, const char *path);
void CErrorCallback(LY_LOG_LEVEL level, const char *msg, const char *path);

struct lyd_node *go_lyd_parse_mem(struct ly_ctx *ctx, const char *data, LYD_FORMAT format, int options);

void printSet(struct ly_set *set);

#endif
