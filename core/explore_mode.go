/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package core

import "context"

// Explore mode (the WikiSkill paper's key lesson: the agent that runs in
// order to FEED the wiki must not read the wiki — grounding on existing
// pages biases it to imitate known answers instead of surfacing what the
// knowledge base is missing). The flag rides the request context so every
// retrieval leg that flows through QueryDocuments honors it; production
// chat never sets it.
//
// Lives in core (not the wiki module) because both modules/document (the
// retrieval filter) and modules/wiki (the future distill/propose jobs that
// set it) need it without an import cycle.

type wikiExploreModeKey struct{}

// WithExploreMode returns a context that marks this request as a
// knowledge-gathering run: wiki-sourced documents are excluded from
// retrieval results.
func WithExploreMode(ctx context.Context) context.Context {
	return context.WithValue(ctx, wikiExploreModeKey{}, true)
}

// ExploreModeEnabled reports whether the context asked for wiki-free
// retrieval. Absent key = production behavior (wiki tier included).
func ExploreModeEnabled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(wikiExploreModeKey{}).(bool)
	return v
}
