export function Welcome(props) {
    const { text } = props;

    if (!text) return null

    return (
        <div className={`w-full text-center color-#999 leading-[24px]`}>
            {text}
        </div>
    )
}

export default Welcome;
