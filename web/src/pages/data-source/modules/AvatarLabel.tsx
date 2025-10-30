import { EntityCard, EntityLabel } from "@infinilabs/entity-ui";
import styles from "./AvatarLabel.module.less";
import { useRequest } from "@sa/hooks";
import { fetchEntityCard } from "@/service/api/entity";

export default function AvatarLabel(props) {
    const { showCard } = props;
    const { type, id } = props.data || {}
    const { hasAuth } = useAuth()
    const permissions = {
        card: hasAuth('generic#entity:card/read')
    }

    const [open, setOpen] = useState(false)

    const { data, loading, run } = useRequest(fetchEntityCard, {
        manual: true,
    });

    const label = (
        <div className={`${styles.label} ${showCard ? 'cursor-pointer' : ''}`} onClick={() => {
            if (showCard) {
                setOpen(!open)
            }
        }}>
            <EntityLabel {...props}/>
        </div>
    )

    useEffect(() => {
        if (open && permissions.card && type && id) {
            run({ type, id })
        }
    }, [open, permissions.card, type, id])
    
    return showCard ? (
        <>
            <EntityCard
                onOpenChange={(open) => setOpen(open)}
                open={open}
                autoPlacement 
                data={loading ? undefined : data}
                trigger={label}
                triggerType="click"
            />
        </>
    ) : label
}