import { cloneDeep, groupBy, keys, map, uniq } from "lodash";
import { fetchBatchShares } from "@/service/api/share";
import { fetchBatchEntityLabels } from "@/service/api/entity";
import { selectUserInfo } from "@/store/slice/auth";

export default function useResource() {

    const userInfo = useAppSelector(selectUserInfo);

    const { hasAuth } = useAuth()

    const permissions = {
        shares: hasAuth('generic#sharing/search'),
        entityLabel: hasAuth('generic#entity:label/read')
    }
    
    const addSharesToData = async (data: any[], resources: any) => {
        if (!Array.isArray(data) || data.length === 0 || !Array.isArray(resources) || resources?.length === 0) return data;
        const newData = cloneDeep(data)
        let shareRes: any;
        if (permissions.shares) {
            shareRes = await fetchBatchShares(resources)
        }
        let entityRes: any
        if (permissions.entityLabel) {
            const entities = newData.filter((item) => !!item._system?.owner_id).map((item) => ({
                type: 'user',
                id: item._system.owner_id
            }))
            if (shareRes?.data?.length > 0) {
                entities.push(...shareRes?.data.map((item) => ({ type: item.principal_type, id: item.principal_id })))
            }
            if (userInfo?.id) {
                entities.push({
                    type: 'user',
                    id: userInfo.id
                })
            }
            const grouped = groupBy(entities, 'type');
            const body = map(keys(grouped), (type) => ({
                type,
                id: uniq(map(grouped[type], 'id')) 
            }))
            entityRes = await fetchBatchEntityLabels(body)
        }
        newData.forEach((item) => {
            const hasEntities = entityRes?.data?.length > 0
            if (hasEntities) {
                if (shareRes?.data?.length > 0) {
                    item.shares = shareRes?.data.filter((s) => s.resource_id === item.id).map((item) => ({
                        ...item,
                        entity: entityRes?.data.find((o) => o.id === item.principal_id)
                    }))
                } else {
                    item.shares = []
                }
                if (item._system?.owner_id) {
                    item.owner = entityRes?.data.find((o) => o.id === item._system?.owner_id)
                }
                if (userInfo?.id) {
                    item.editor = entityRes?.data.find((o) => o.id === userInfo?.id)
                }
            }
        })
        return newData
    } 

    function isEditorOwner(record) {
        return record?.owner?.id && record?.owner?.id === record?.editor?.id
    }

    function hasEdit(record) {
        const share = record?.shares?.find((item) => item.principal_id === record?.editor?.id)
        return isEditorOwner(record) || share?.permission >= 4
    }

    function hasView(record) {
        const share = record?.shares?.find((item) => item.principal_id === record?.editor?.id)
        return isEditorOwner(record) || share?.permission >= 1
    }

    function isResourceShare(record) {
        return record?.owner && record?.owner?.id !== record?.editor?.id
    }

    return {
        addSharesToData,
        isEditorOwner,
        hasEdit,
        hasView,
        isResourceShare
    }
}