import { Avatar, Button, Popover } from "antd";
import { UserOutlined } from '@ant-design/icons';
import AvatarLabel from "./AvatarLabel";
import AddShares from "./AddShares";
import EditShares from "./EditShares";

export const PERMISSION_MAPPING = {
    1: 'view',
    2: 'comment',
    4: 'edit',
    8: 'share',
    16: 'all'
}

export default (props) => {

    const { title, record = {}, resourceType, resourceID, resourcePath, onSuccess } = props;

    const { t } = useTranslation();

    const { owner, shares = [], editor } = record

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
    const [isAdding, setIsAdding] = useState(false);

    const handleOpenChange = (newOpen: boolean) => {
        setOpen(newOpen);
        if (newOpen === false) {
            setIsAdding(false)
        }
    };
    
    const handleSuccess = () => {
        setOpen(false)
        setIsAdding(false)
        onSuccess && onSuccess()
    }

    const hasShardPermission = useMemo(() => {
        if (owner?.id === editor?.id) return true;
        const permission = shares.find((item) => item.principal_id === editor?.id)
        return permission ? [8, 16].includes(permission) : false
    }, [owner, editor, shares])

    const content = isAdding && hasShardPermission ? (
        <AddShares 
            permissions={permissions} 
            onCancel={() => handleOpenChange(false)} 
            onSuccess={handleSuccess}
            resourceType={resourceType}
            resourceID={resourceID}
            resourcePath={resourcePath}
            owner={owner} 
            editor={editor}
            shares={shares}
        />
    ) : (
        <EditShares 
            hasEdit={hasShardPermission} 
            permissions={permissions} 
            owner={owner} 
            editor={editor}
            shares={shares} 
            onCancel={() => handleOpenChange(false)} 
            onAddShares={() => setIsAdding(true)} 
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
                    <Button className="px-0" type="link" onClick={() => setIsAdding(true)}>{t('common.add')}</Button>
                ) : (
                    <div>
                        <Avatar.Group max={{ count: 5 }} size={"small"}>
                            {
                                shares.map((item, index) => (
                                    item.avatar ? (
                                        <Avatar key={index} src={<img draggable={false} src={item.avatar} />} />
                                    ) : (
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