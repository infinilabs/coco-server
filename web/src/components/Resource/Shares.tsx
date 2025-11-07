import { Avatar, Button, Popover, Typography } from "antd";
import { UserOutlined } from '@ant-design/icons';
import AvatarLabel from "./AvatarLabel";
import AddShares from "./AddShares";
import EditShares from "./EditShares";
import { differenceBy } from "lodash";
import { updateShares } from "@/service/api/share";

export const PERMISSION_MAPPING = {
    0: 'none',
    1: 'view',
    2: 'comment',
    4: 'edit',
    8: 'share',
    16: 'all'
}

export default (props) => {

    const { title, record = {}, resource, onSuccess } = props;

    const { t } = useTranslation();

    const { owner, shares = [], editor } = record

    const { hasAuth } = useAuth()

    const permissions = {
        create: hasAuth('generic#sharing/create'),
        update: hasAuth('generic#sharing/update'),
    }

    const permissionOptions = [
        {
            key: 0,
            label: t(`page.datasource.labels.${PERMISSION_MAPPING[0]}`)
        },
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
    ]
    const [open, setOpen] = useState(false);
    const [lockOpen, setLockOpen] = useState(false);
    const [isAdding, setIsAdding] = useState(false);

    const [currentShares, setCurrentShares] = useState([]);

    const initShares = (shares, editor) => {
        setCurrentShares(shares.filter((item) => item.principal_id !== editor?.id))
    }

    useEffect(() => {
        initShares(shares, editor)
    }, [shares, editor])

    const handleOpenChange = (newOpen: boolean) => {
        if (lockOpen) return;
        setOpen(newOpen);
        if (newOpen === false) {
            initShares(shares, editor)
            setIsAdding(false)
        }
    };

    const handleUpdate = async (currentShares) => {
        const sourceShares = shares.filter((item) => item.principal_id !== editor?.id);
        if (JSON.stringify(sourceShares) !== JSON.stringify(currentShares)) {
            const deletedItems = differenceBy(sourceShares, currentShares, 'principal_id');
            const res = await updateShares({
                type: resource?.resource_type,
                id: resource?.resource_id,
                shares: currentShares.filter((item) => item.via !== 'inherit' || item.removeVia).map((item) => ({
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
                handleSuccess()
            }
        } else {
            handleOpenChange(false)
        }
    }
    
    const handleSuccess = () => {
        setOpen(false)
        setIsAdding(false)
        onSuccess && onSuccess()
    }

    const hasSharePermission = useMemo(() => {
        if (owner?.id === editor?.id) return true;
        const share = shares.find((item) => item.principal_id === editor?.id)
        return share?.permission >= 8
    }, [owner, editor, shares])

    const hasCreate = permissions.create && hasSharePermission
    const hasEdit = permissions.update && hasSharePermission

    const content = isAdding ? (
        <AddShares 
            hasCreate={hasCreate}
            permissionOptions={permissionOptions} 
            onCancel={() => handleOpenChange(false)} 
            onSuccess={handleSuccess}
            resource={resource}
            owner={owner} 
            editor={editor}
            shares={shares}
            currentShares={currentShares}
            onChange={setCurrentShares}
            onSubmit={handleUpdate}
            setLockOpen={setLockOpen}
        />
    ) : (
        <EditShares
            hasCreate={hasCreate}
            hasEdit={hasEdit} 
            permissionOptions={permissionOptions} 
            owner={owner} 
            editor={editor}
            shares={shares} 
            onCancel={() => handleOpenChange(false)} 
            onAddShares={() => setIsAdding(true)} 
            currentShares={currentShares}
            onChange={setCurrentShares}
            onSubmit={handleUpdate}
            setLockOpen={setLockOpen}
        />
    )
    
    if (shares.length === 0 && !hasCreate) {
        return '-'
    }

    return (
        <Popover 
            trigger={'click'} 
            placement="bottom"
            title={(
                <Typography.Paragraph
                    ellipsis={{
                        rows: 1,
                        tooltip: true,
                    }}
                    className="text-16px mb-16px"
                >
                    {`${t('page.datasource.labels.shareTo')} ${title}`}
                </Typography.Paragraph>
            )} 
            open={open || lockOpen}
            onOpenChange={handleOpenChange}
            content={content}
            className="flex w-fit cursor-pointer"
            rootClassName={"w-362px"}
            destroyTooltipOnHide
        >
            {
                shares.length === 0 ? (
                    <Button className="px-0" type="link" onClick={() => setIsAdding(true)}>{t('common.add')}</Button>
                ) : (
                    <div>
                        <Avatar.Group max={{ count: 5 }} size={"small"} className="items-center">
                            {
                                shares.map((item, index) => (
                                    item.entity ? (
                                        <AvatarLabel
                                            key={index} 
                                            data={{
                                                type: item.entity.type,
                                                id: item.entity.id,
                                                icon: item.entity.icon,
                                            }}
                                        />
                                    ) : (
                                        <Avatar key={index} size={"small"} icon={<UserOutlined />} />
                                    )
                                ))
                            }
                        </Avatar.Group>
                    </div>
                )
            }
        </Popover>
    )
}