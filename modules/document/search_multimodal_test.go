/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package document

import (
	"context"
	"strings"
	"testing"

	"infini.sh/coco/core"
)

func attachmentWith(id string, owner string, text string) *core.Attachment {
	att := core.Attachment{}
	att.ID = id
	att.Text = text
	if owner != "" {
		att.SetOwnerID(owner)
	}
	return &att
}

func repeats(s string, n int) string {
	return strings.Repeat(s, n)
}

// the multimodal merge rules: which attachment text joins the query, how the
// caps apply, and what the Warning-header note says when something is skipped
func TestEnrichQueryMultimodalMerge(t *testing.T) {
	noopWait := func(ctx context.Context, ids []string) error { return nil }
	waitErr := func(ctx context.Context, ids []string) error { return context.DeadlineExceeded }

	cases := []struct {
		name       string
		query      string
		param      string
		userID     string
		wait       func(ctx context.Context, ids []string) error
		load       func(ids []string) []*core.Attachment
		wantQuery  string
		wantNoteIs string // note substring that must be present
		wantNoteNo bool   // note must stay empty
	}{
		{
			name:       "no attachments param is a no-op",
			query:      "hello",
			param:      "",
			load:       func(ids []string) []*core.Attachment { t.Fatal("load must not run"); return nil },
			wantQuery:  "hello",
			wantNoteNo: true,
		},
		{
			name:       "blank ids are dropped without a load",
			param:      " , ,",
			load:       func(ids []string) []*core.Attachment { t.Fatal("load must not run"); return nil },
			wantNoteNo: true,
		},
		{
			name:   "image-only search: text becomes the query",
			param:  "att1",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{attachmentWith("att1", "u1", "a blue circuit board close-up")}
			},
			wantQuery:  "a blue circuit board close-up",
			wantNoteNo: true,
		},
		{
			name:   "typed query keeps its place before attachment text",
			query:  "circuit board",
			param:  "att1",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{attachmentWith("att1", "u1", "blue PCB photo")}
			},
			wantQuery:  "circuit board blue PCB photo",
			wantNoteNo: true,
		},
		{
			name:   "wait timeout still searches the ready text",
			param:  "att1",
			userID: "u1",
			wait:   waitErr,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{attachmentWith("att1", "u1", "extracted")}
			},
			wantQuery:  "extracted",
			wantNoteNo: true,
		},
		{
			name:   "missing/empty-text attachments are counted in the note",
			param:  "att1,att2,att3",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{
					attachmentWith("att1", "u1", ""),               // extraction pending
					attachmentWith("att3", "someone-else", "text"), // not the requester's
				}
			},
			wantQuery:  "",
			wantNoteIs: "2 of 3 attachments",
		},
		{
			name:   "partial merge: ready text joins, skipped ones note",
			query:  "photo",
			param:  "att1,att2",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{
					attachmentWith("att1", "u1", "sunset"),
					attachmentWith("att2", "u1", ""),
				}
			},
			wantQuery:  "photo sunset",
			wantNoteIs: "1 of 2 attachments",
		},
		{
			name:   "legacy attachments without an owner are searchable",
			param:  "att1",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{attachmentWith("att1", "", "ownerless text")}
			},
			wantQuery:  "ownerless text",
			wantNoteNo: true,
		},
		{
			name:   "per-attachment cap bounds each contribution",
			param:  "att1",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				return []*core.Attachment{attachmentWith("att1", "u1", repeats("x", multimodalAttachmentTextCap+50))}
			},
			wantQuery:  repeats("x", multimodalAttachmentTextCap),
			wantNoteNo: true,
		},
		{
			name:   "combined cap bounds the merged text",
			param:  "att1,att2,att3,att4",
			userID: "u1",
			wait:   noopWait,
			load: func(ids []string) []*core.Attachment {
				out := []*core.Attachment{}
				for _, id := range ids {
					out = append(out, attachmentWith(id, "u1", repeats("y", multimodalAttachmentTextCap)))
				}
				return out
			},
			// 4x400 joined with single spaces = 1603 chars; the combined cap
			// keeps the first 1000 including the two separator spaces
			wantQuery:  repeats("y", multimodalAttachmentTextCap) + " " + repeats("y", multimodalAttachmentTextCap) + " " + repeats("y", multimodalQueryTextCap-2*multimodalAttachmentTextCap-2),
			wantNoteNo: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wait := c.wait
			if wait == nil {
				wait = noopWait
			}
			gotQuery, gotNote := enrichQuery(context.Background(), c.query, c.param, c.userID, wait, c.load)

			if gotQuery != c.wantQuery {
				t.Fatalf("query = %q (len %d), want %q (len %d)", gotQuery, len(gotQuery), c.wantQuery, len(c.wantQuery))
			}
			if c.wantNoteIs != "" && !strings.Contains(gotNote, c.wantNoteIs) {
				t.Fatalf("note = %q, want it to contain %q", gotNote, c.wantNoteIs)
			}
			if c.wantNoteNo && gotNote != "" {
				t.Fatalf("note should stay empty, got %q", gotNote)
			}
		})
	}
}
