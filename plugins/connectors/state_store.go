package connectors

import (
	"context"
	"fmt"

	"infini.sh/coco/modules/common"
	"infini.sh/framework/core/orm"
)

const syncStateIndexName = "connector_sync_state"

func init() {
	suffix := common.GetSchemaSuffix()
	orm.MustRegisterSchemaWithIndexName(SyncState{}, syncStateIndexName+suffix)
}

type StoredCursorValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type StoredCursor struct {
	Property StoredCursorValue  `json:"property"`
	Tie      *StoredCursorValue `json:"tie,omitempty"`
}

type SyncState struct {
	orm.ORMObjectBase
	ConnectorID  string        `json:"connector_id" elastic_mapping:"connector_id:{type:keyword}"`
	DatasourceID string        `json:"datasource_id" elastic_mapping:"datasource_id:{type:keyword}"`
	Mode         string        `json:"mode,omitempty" elastic_mapping:"mode:{type:keyword}"`
	Property     string        `json:"property,omitempty" elastic_mapping:"property:{type:keyword}"`
	Cursor       *StoredCursor `json:"cursor,omitempty" elastic_mapping:"cursor:{enabled:false}"`
}

type SyncStateStore struct{}

func NewSyncStateStore() *SyncStateStore {
	return &SyncStateStore{}
}

func (s *SyncStateStore) Load(ctx context.Context, connectorID, datasourceID string) (*SyncState, error) {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.DirectReadAccess()

	state := &SyncState{}
	state.SetID(makeSyncStateID(connectorID, datasourceID))

	exists, err := orm.GetV2(ormCtx, state)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return state, nil
}

func (s *SyncStateStore) Save(ctx context.Context, state *SyncState) error {
	if state == nil {
		return nil
	}
	if state.GetID() == "" {
		state.SetID(makeSyncStateID(state.ConnectorID, state.DatasourceID))
	}
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.DirectReadAccess()
	ormCtx.Refresh = orm.WaitForRefresh
	return orm.Save(ormCtx, state)
}

func (s *SyncStateStore) Clear(ctx context.Context, connectorID, datasourceID string) error {
	ormCtx := orm.NewContextWithParent(ctx)
	ormCtx.DirectReadAccess()
	state := &SyncState{}
	state.SetID(makeSyncStateID(connectorID, datasourceID))
	exists, err := orm.GetV2(ormCtx, state)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	return orm.Delete(ormCtx, state)
}

func makeSyncStateID(connectorID, datasourceID string) string {
	return fmt.Sprintf("%s:%s", connectorID, datasourceID)
}
