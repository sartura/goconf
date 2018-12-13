/**
 * @file helper.c
 * @author Mislav Novakovic <mislav.novakovic@sartura.hr>
 * @brief implementation of helper function for go program.
 *
 * @copyright
 * Copyright 2017 Deutsche Telekom AG.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#include <stdbool.h>

#include <libyang/libyang.h>
#include <libyang/tree_data.h>
#include "helper.h"
#include "_cgo_export.h"

void CErrorCallback(LY_LOG_LEVEL level, const char *msg, const char *path) {
	return GoErrorCallback(level, (char *) msg, (char *) path);
}

struct lyd_node *go_lyd_parse_mem(struct ly_ctx *ctx, const char *data, LYD_FORMAT format, int options) {
	return lyd_parse_mem(ctx, data, format, options);
}

void printXPATH(struct lyd_node *node) {
	struct lyd_node *elem = NULL, *next = NULL;

	LY_TREE_DFS_BEGIN(node, next, elem) {
		if (LYS_LEAF == elem->schema->nodetype || LYS_LEAFLIST == elem->schema->nodetype) {
			char *stringXpath = lyd_path(elem);
			if (NULL == stringXpath) {
				return;
			}
			printf("%s %s\n", stringXpath, ((struct lyd_node_leaf_list *) elem)->value_str);
			free(stringXpath);
		}
	LY_TREE_DFS_END(node, next, elem) }
}

void printSet(struct ly_set *set) {
	for (int i = 0; i < set->number; i++) {
		printXPATH(set->set.d[i]);
	}
}

const char *get_features(struct lys_feature *features, int i) {
    if (!features) return NULL;
    return features[i].name;
};
