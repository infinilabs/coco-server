import styles from "./Loading.module.less";
import { Spin } from "antd";

export default (props) => {
    const { loading = true, currentLocales } = props;
    if (!loading) return null;
    return (
        <div className={styles.loading}>
            <Spin tip={currentLocales["dropdownlist.loading"]}/>
        </div>
    )
}