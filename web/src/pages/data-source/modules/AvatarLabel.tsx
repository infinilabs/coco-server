import { EntityCard, EntityLabel } from "@infinilabs/entity-ui";
import styles from "./AvatarLabel.module.less";

export default function AvatarLabel(props) {
    return (
        <div className={styles.label}>
            <EntityLabel {...props}/>
        </div>
    )
}