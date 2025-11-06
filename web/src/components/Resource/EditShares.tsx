import { cloneDeep } from "lodash";
import AvatarLabel from "./AvatarLabel";
import { Button, Dropdown, Space } from "antd";
import { DownOutlined, MinusCircleOutlined, UsergroupAddOutlined } from '@ant-design/icons';
import styles from './EditShares.module.less'
import { PERMISSION_MAPPING } from "./Shares";

export default function EditShares(props) {

    const { hasCreate, hasEdit, permissionOptions = [], owner, shares = [], editor, onCancel, onAddShares, currentShares, onChange, onSubmit, setLockOpen } = props;
    const { t } = useTranslation();

    const handleChange = (index, permission) => {
        const newData = cloneDeep(currentShares);
        if (newData[index]) {
            if (newData[index].via === 'inherit') {
                newData[index].removeVia = true
            }
            newData[index].permission = parseInt(permission)
            onChange(newData)
        }
    }

    const handleDelete = (index) => {
        const newData = cloneDeep(currentShares);
        newData.splice(index, 1)
        onChange(newData)
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
                    !isOwner && editor && Number.isInteger(editorPermission?.permission) && (
                        <div className={styles.item}>
                            <AvatarLabel
                                data={{
                                    ...editor,
                                    title: `${editor.title}${t('page.datasource.labels.you')}${editorPermission?.via === 'inherit' ? t('page.datasource.labels.inherit') : ''}`,
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
                    currentShares.filter((item) => !!item.entity).map((item: any, index) => {
                        const isInherit = item.via === 'inherit'
                        return (
                            <div key={index} className={styles.item}>
                                <AvatarLabel
                                    data={{
                                        ...item.entity,
                                        title: `${item.entity.title}${isInherit ? t('page.datasource.labels.inherit') : ''}`
                                    }}
                                />
                                <div className={styles.actions}>
                                    {
                                        hasEdit ? (
                                            <Space>
                                                <Dropdown 
                                                    trigger={['click']} 
                                                    onOpenChange={setLockOpen}
                                                    menu={{ 
                                                        items: permissionOptions, 
                                                        onClick: ({key}) => handleChange(index, key)  
                                                    }}>
                                                    <Button size="small" className="px-6px text-12px" type="text">{Number.isInteger(item.permission) ? t(`page.datasource.labels.${PERMISSION_MAPPING[item.permission]}`) : ''}<DownOutlined /></Button>
                                                </Dropdown>
                                                {
                                                    !isInherit && (<MinusCircleOutlined className="cursor-pointer" onClick={() => handleDelete(index)}/>)
                                                }
                                            </Space>
                                        ) : (
                                            <span className="text-[var(--ant-color-text-secondary)]">{Number.isInteger(item.permission) ? t(`page.datasource.labels.${PERMISSION_MAPPING[item.permission]}`) : ''}</span>
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
                                    <Button className="w-80px" type="primary" onClick={() => onSubmit(currentShares)}>
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