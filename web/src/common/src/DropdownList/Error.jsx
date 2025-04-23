import styles from "./Error.module.less";
import Icon, { CloseCircleOutlined } from "@ant-design/icons";

export default (props) => {
    const { currentLocales, failed = true } = props;
    if (!failed) return null;
    return (
        <div className={styles.error}>
            <Icon component={CloseCircleOutlined} />
            <div className={styles.tips}>{currentLocales["dropdownlist.loading.failed"]}</div>
        </div>
    )
}