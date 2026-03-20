import { Spin } from "antd"

export default (props) => {

    const { theme, loading } = props;

    return loading ? (
        <div style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: theme === 'dark' ? 'rgba(0, 0, 0, 0.85)' : 'rgba(255, 255, 255, 0.85)',
            zIndex: 99999,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            pointerEvents: 'auto',
            backdropFilter: 'blur(2px)',
        }}>
            <Spin />
        </div>
    ) : null
}