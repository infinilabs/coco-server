import { updateShares } from "@/service/api/share";
import { cloneDeep, differenceBy } from "lodash";
import AvatarLabel from "./AvatarLabel";
import { Button, Dropdown, Space } from "antd";
import { DownOutlined, MinusCircleOutlined, UsergroupAddOutlined } from '@ant-design/icons';
import styles from './EditShares.module.less'
import { PERMISSION_MAPPING } from "./Shares";

export default function EditShares(props) {

    const { hasCreate, hasEdit, resource, permissionOptions = [], owner, shares = [], editor, onCancel, onAddShares, onSuccess } = props;
    const { t } = useTranslation();

    const [currentData, setCurrentData] = useState([]);

    useEffect(() => {
        setCurrentData(shares.filter((item) => item.principal_id !== editor?.id))
    }, [shares])

    const handleChange = (index, permission) => {
        const newData = cloneDeep(currentData);
        if (newData[index]) {
            if (newData[index].via === 'inherit') {
                delete newData[index].via
            }
            newData[index].permission = parseInt(permission)
            setCurrentData(newData)
        }
    }

    const handleDelete = (index) => {
        const newData = cloneDeep(currentData);
        newData.splice(index, 1)
        setCurrentData(newData)
    }

    const handleUpdate = async () => {
        const sourceData = shares.filter((item) => item.principal_id !== editor?.id);
        if (JSON.stringify(sourceData) !== JSON.stringify(currentData)) {
            const deletedItems = differenceBy(sourceData, currentData, 'principal_id');
            const res = await updateShares({
                type: resource?.resource_type,
                id: resource?.resource_id,
                shares: currentData.filter((item) => item.via !== 'inherit').map((item) => ({
                    ...(resource || {}),
                    "principal_type": "user",
                    "principal_id": item.principal_id,
                    permission: item.permission,
                })),
                revokes: (deletedItems || []).map((item) => ({
                    "id": item.id,
                    "principal_type": "user",
                    "principal_id": item.principal_id,
                    permission: item.permission
                }))
            })
            if (res && !res.error) {
                window.$message?.success(t('common.updateSuccess'));
                onSuccess && onSuccess()
            }
        }
    }

    const renderSpecialItems = () => {
        const isOwner = editor?.id === owner?.id;
        const editorPermission = shares.find((item) => item.principal_id === editor?.id)
        return (
            <>
                {
                    owner && (
                        <div className={styles.item}>
                            <AvatarLabel
                                data={{
                                    ...owner,
                                    title: `${owner.title}${isOwner ? t('page.datasource.labels.you') : ''}`,
                                }}
                            />
                            <div className={styles.actions}>
                                <span className="text-[var(--ant-color-text-secondary)]">{t('page.datasource.labels.owner')}</span>
                            </div>
                        </div>
                    )
                }
                {
                    !isOwner && editor && editorPermission?.permission && (
                        <div className={styles.item}>
                            <AvatarLabel
                                data={{
                                    ...editor,
                                    title: `${editor.title}${t('page.datasource.labels.you')} ${editorPermission?.via === 'inherit' ? '(Inherit)' : ''}`,
                                }}
                            />
                            <div className={styles.actions}>
                                <span className="text-[var(--ant-color-text-secondary)]">
                                    {t(`page.datasource.labels.${PERMISSION_MAPPING[editorPermission.permission]}`)}
                                </span>
                            </div>
                        </div>
                    )
                }
            </>
        )
    }

    return (
        <div >
            <div className="text-14px mb-8px">{t('page.datasource.labels.sharesWithPermissions')}</div>
            <div className={`max-h-278px ${hasEdit || hasCreate ? 'mb-24px' : 'mb-0'} border border-[var(--ant-color-border)] rounded-[var(--ant-border-radius)] px-8px py-12px overflow-auto`}>
                {renderSpecialItems()}
                {
                    currentData.filter((item) => !!item.entity).map((item, index) => {
                        const isInherit = item.via === 'inherit'
                        return (
                            <div key={index} className={styles.item}>
                                <AvatarLabel
                                    data={{
                                        ...item.entity,
                                        title: `${item.entity.title}${isInherit ? '(Inherit)' : ''}`
                                    }}
                                />
                                <div className={styles.actions}>
                                    {
                                        hasEdit ? (
                                            <Space>
                                                <Dropdown trigger={['click']} menu={{ items: permissionOptions.filter((item) => {
                                                    if (isInherit) return true;
                                                    return item.key > 0
                                                }), onClick: ({key}) => handleChange(index, key)  }}>
                                                    <Button size="small" className="px-6px text-12px" type="text">{item.permission ? t(`page.datasource.labels.${PERMISSION_MAPPING[item.permission]}`) : ''}<DownOutlined /></Button>
                                                </Dropdown>
                                                {
                                                    !isInherit && (<MinusCircleOutlined className="cursor-pointer" onClick={() => handleDelete(index)}/>)
                                                }
                                            </Space>
                                        ) : (
                                            <span className="text-[var(--ant-color-text-secondary)]">{item.permission ? t(`page.datasource.labels.${PERMISSION_MAPPING[item.permission]}`) : ''}</span>
                                        )
                                    }
                                </div>
                            </div>
                        )
                    })
                }
            </div>
            {
                (hasCreate || hasEdit) && (
                    <div className={`flex items-center ${hasCreate ? 'justify-between' : 'justify-right'}`}>
                        {
                            hasCreate && (
                                <Button className="w-80px" type="primary" ghost onClick={() => onAddShares()}>
                                    <UsergroupAddOutlined />
                                </Button>
                            )
                        }
                        {
                            hasEdit && (
                                <Space>
                                    <Button className="w-80px" type="primary" ghost onClick={() => onCancel()}>
                                        {t('common.cancel')}
                                    </Button>
                                    <Button className="w-80px" type="primary" onClick={() => handleUpdate()}>
                                        {t('common.ok')}
                                    </Button>
                                </Space>
                            )
                        }
                    </div>
                )
            }
        </div>
    )
}