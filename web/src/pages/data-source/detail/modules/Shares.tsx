import { Avatar, Button, Dropdown, Form, message, Popover, Select, Space } from "antd";
import { DownOutlined, MinusCircleOutlined, UsergroupAddOutlined } from '@ant-design/icons';
import { EntityLabel } from '@infinilabs/entity-ui';
import styles from './Shares.module.less'
import PrincipalSelect from "./PrincipalSelect";
import { addShares, fetchCurrentUserPermission, updateShares } from "@/service/api/share";
import { cloneDeep } from "lodash";

const PERMISSION_MAPPING = {
    1: 'view',
    2: 'comment',
    4: 'edit',
    8: 'share',
    16: 'all'
}

export default (props) => {

    const { datasource, title, record = {}, resourceType, resourceID, resourcePath, onSuccess } = props;

    const { t } = useTranslation();

    const { owner = {},  shares = [] } = record

    const permissions = [
        {
            key: 1,
            label: t(`page.datasource.labels.${PERMISSION_MAPPING[1]}`)
        },
        {
            key: 2,
            label: t(`page.datasource.labels.${PERMISSION_MAPPING[2]}`)
        },
        {
            key: 4,
            label: t(`page.datasource.labels.${PERMISSION_MAPPING[4]}`)
        },
        {
            key: 8,
            label: t(`page.datasource.labels.${PERMISSION_MAPPING[8]}`)
        },
        {
            key: 16,
            label: t(`page.datasource.labels.${PERMISSION_MAPPING[16]}`)
        }
    ]
    const [open, setOpen] = useState(false);
    const [isEdit, setIsEdit] = useState(false);
    const [permission, setPermission] = useState();

    const fetchPermission = async (record, datasource) => {
        if (!datasource?.connector?.id || !record?.id) return;
        const res = await fetchCurrentUserPermission({
            type: datasource.connector.id,
            id: record.id
        })
        if (res && !res.error) {
            setPermission(res)
        }
    }

    const handleOpenChange = (newOpen: boolean) => {
        setOpen(newOpen);
        if (newOpen === false) {
            setIsEdit(false)
        }
    };
    
    const handleSuccess = () => {
        setOpen(false)
        onSuccess && onSuccess()
    }

    useEffect(() => {
        if (open) {
            // fetchPermission(record, datasource)
        }
    }, [open, datasource, record])

    const hasEdit = useMemo(() => {
        // return permission && ['share', 'owner'].includes(permission)
        return true;
    }, [permission])

    const content = isEdit && hasEdit ? (
        <EditShares 
            permissions={permissions} 
            onCancel={() => handleOpenChange(false)} 
            onSuccess={handleSuccess}
            resourceType={resourceType}
            resourceID={resourceID}
            resourcePath={resourcePath}
        />
    ) : (
        <Shares 
            hasEdit={hasEdit} 
            permissions={permissions} 
            owner={owner} 
            shares={shares} 
            onCancel={() => handleOpenChange(false)} 
            onEditShares={() => setIsEdit(true)} 
            onSuccess={handleSuccess}
            resourceType={resourceType}
            resourceID={resourceID}
            resourcePath={resourcePath}
        />
    )

    return (
        <Popover 
            trigger={'click'} 
            placement="bottom"
            title={<div className="text-16px mb-16px">{`${t('page.datasource.labels.share')} ${title}`}</div>} 
            open={open}
            onOpenChange={handleOpenChange}
            content={content}
            className="flex w-fit cursor-pointer"
            rootClassName={"min-w-362px"}
            destroyTooltipOnHide
        >
            {
                shares.length === 0 ? (
                    <Button type="link" onClick={() => setIsEdit(true)}>{t('common.add')}</Button>
                ) : (
                    <div>
                        <Avatar.Group max={{ count: 5 }} size={"small"} shape="circle">
                            {
                                shares.map((item, index) => <Avatar key={index} src={<img draggable={false} src={item.avatar} />} />)
                            }
                        </Avatar.Group>
                    </div>
                )
            }
        </Popover>
    )
}

const EditShares = (props) => {

    const { permissions = [], onCancel, onSuccess, resourceType, resourceID, resourcePath } = props;

    const { t } = useTranslation();
    const [form] = Form.useForm();

    const onFinish = async (values) => {
        const { permission, shares = [] } = values;
        const formatShares = shares.map((item) => {
            const share = {
                "resource_type": resourceType,
                "resource_id": resourceID,
                "principal_type": "user",
                "principal_id": item.id,
                permission,
            }
            if (resourcePath) {
                share['resource_path'] = resourcePath
            }
            return share
        })
        const res = await addShares({ shares: formatShares })
        if (res?.data?.created) {
            message.success(t('common.addSuccess'));
            onSuccess && onSuccess()
        }
    }

    return (
        <div>
            <Form
                colon={false}
                form={form}
                layout="vertical"
                onFinish={onFinish}
            >
                <Form.Item
                    label={t('page.datasource.labels.shareTo')}
                    name="shares"
                >
                    <PrincipalSelect mode="multiple"/>
                </Form.Item>
                <Form.Item
                    label={t('page.datasource.labels.permission')}
                    name="permission"
                >
                    <Select options={permissions.map((item) => ({ ...item, value: item.key}))}/>
                </Form.Item>
                <Form.Item className="mb-12px">
                    <div className="flex items-center justify-right">
                        <Space>
                            <Button className="w-80px" type="primary" ghost onClick={() => onCancel()}>
                                {t('common.cancel')}
                            </Button>
                            <Button className="w-80px" type="primary" htmlType="submit">
                                {t('common.ok')}
                            </Button>
                        </Space>
                    </div>
                </Form.Item>
            </Form>
        </div>
    )
}

const Shares = (props) => {

    const { hasEdit, permissions = [], owner, shares = [], onCancel, onEditShares, onSuccess } = props;
    const { t } = useTranslation();

    const [currentData, setCurrentData] = useState([]);

    useEffect(() => {
        setCurrentData(shares)
    }, [shares])

    const handleChange = (index, permission) => {
        const newData = cloneDeep(currentData);
        if (newData[index]) {
            newData[index].permission = permission
            setCurrentData(newData)
        }
    }

    const handleDelete = (index) => {
        const newData = cloneDeep(currentData);
        newData.splice(index, 1)
        setCurrentData(newData)
    }

    const handleUpdate = async () => {
        if (JSON.stringify(data) !== JSON.stringify(currentData)) {
            const res = await updateShares(currentData.map((item) => ({
                "principal_type": "user",
                "principal_id": item.id,
                permission: item.permission
            })))
            if (res?.data?.updated) {
                message.success(t('common.updateSuccess'));
                onSuccess && onSuccess()
            }
        }
    }

    return (
        <div >
            <div className="text-14px mb-8px">{t('page.datasource.labels.sharesWithPermissions')}</div>
            <div className={`max-h-278px ${hasEdit ? 'mb-24px' : 'mb-0'} border border-[var(--ant-color-border)] rounded-[var(--ant-border-radius)] px-8px py-12px overflow-auto`}>
                <div className={styles.item}>
                    <div className={styles.label}>
                        <EntityLabel
                            data={{
                                type: 'user',
                                id: 'svc-1',
                                icon: owner.avatar,
                                title: owner.nickname,
                                subtitle: owner.email,
                            }}
                        />
                    </div>
                    <div className={styles.actions}>
                        <span className="text-[var(--ant-color-text-secondary)]">{t('page.datasource.labels.owner')}</span>
                    </div>
                </div>
                {
                    currentData.map((item, index) => (
                        <div key={index} className={styles.item}>
                            <div className={styles.label}>
                                <EntityLabel
                                    data={{
                                        type: 'user',
                                        id: 'svc-1',
                                        icon: item.avatar,
                                        title: item.nickname,
                                        subtitle: item.email,
                                    }}
                                />
                            </div>
                            <div className={styles.actions}>
                                {
                                    hasEdit ? (
                                        <Space>
                                            <Dropdown trigger={['click']} menu={{ items: permissions, onClick: ({key}) => handleChange(index, key)  }}>
                                                <Button className="px-6px" type="text">{item.permission ? t(`page.datasource.labels.${PERMISSION_MAPPING[item.permission]}`) : ''}<DownOutlined /></Button>
                                            </Dropdown>
                                            <MinusCircleOutlined className="cursor-pointer" onClick={() => handleDelete(index)}/>
                                        </Space>
                                    ) : (
                                        <span className="text-[var(--ant-color-text-secondary)]">{item.permission ? t(`page.datasource.labels.${PERMISSION_MAPPING[item.permission]}`) : ''}</span>
                                    )
                                }
                            </div>
                        </div>
                    ))
                }
            </div>
            {
                hasEdit && (
                    <div className="flex items-center justify-between">
                        <Button className="w-80px" type="primary" ghost onClick={() => onEditShares()}>
                            <UsergroupAddOutlined />
                        </Button>
                        <Space>
                            <Button className="w-80px" type="primary" ghost onClick={() => onCancel()}>
                                {t('common.cancel')}
                            </Button>
                            <Button className="w-80px" type="primary" onClick={() => handleUpdate()}>
                                {t('common.ok')}
                            </Button>
                        </Space>
                    </div>
                )
            }
        </div>
    )
}