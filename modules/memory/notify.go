/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

package memory

import (
	"fmt"

	log "github.com/cihub/seelog"
	"infini.sh/coco/core"
	"infini.sh/framework/core/orm"
)

// Pending-memory notification (W6 close-out): the confirmation gate only
// works if the owner knows there's something to confirm. Distillation
// files records silently — without this, the pending queue is a room
// nobody walks into. The notification lands in the wiki-notification
// store (the console's shared notification surface) with a neutral
// action so the wiki module's read path picks it up.

// NotifyPendingMemories writes one notification per distill run (not per
// record — a batch of five is one "you have N new memories" ping).
// Best-effort: notification failure never fails the distillation.
func NotifyPendingMemories(userID string, count int) {
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("memory: pending notification skipped (store unavailable): %v", r)
		}
	}()
	if userID == "" || count <= 0 {
		return
	}

	ctx := orm.NewContext()
	ctx.Set(orm.DirectWriteWithoutPermissionCheck, true)
	ctx.Refresh = orm.WaitForRefresh
	orm.WithModel(ctx, &core.WikiNotification{})

	notification := &core.WikiNotification{
		UserID:     userID,
		TargetType: "memory",
		TargetID:   userID,
		Action:     "memory-pending",
		Message:    fmt.Sprintf("知识助手从对话中提炼了 %d 条待确认记忆 — 在 设置 → 我的记忆 中确认或拒绝", count),
	}
	if err := orm.Create(ctx, notification); err != nil {
		log.Debugf("memory: pending notification create failed: %v", err)
	}
}
