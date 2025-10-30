import { EntityCard, EntityLabel } from "@infinilabs/entity-ui";
import styles from "./AvatarLabel.module.less";

export default function AvatarLabel(props) {
    return (
        <EntityCard
          triggerType="hover" 
          hoverOpenDelay={500} 
          autoPlacement 
          {...props}
          trigger={(
            <div className={styles.label}>
                <EntityLabel {...props}/>
             </div>
            )}
        />
    )
}