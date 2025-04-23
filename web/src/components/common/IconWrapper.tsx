const Icon = (props) => {

    const { className = '', children } = props

    return (
        <div className={`overflow-hidden bg-#fff border border-[var(--ant-color-border)] rounded-[50%] flex items-center justify-center ${className}`}>
            {children}
        </div>
    )
};

export default Icon;
