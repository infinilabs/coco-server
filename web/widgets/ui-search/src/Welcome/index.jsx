export function Welcome(props) {
    const { text } = props;

    if (!text) return null

    return (
        <div className={`w-full text-center text-16px text-#333 dark:text-#666 leading-[24px]`}>
            {text}
        </div>
    )
}

export default Welcome;
