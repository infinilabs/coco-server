
export function Logo(props) {
    const { isFirst, isMobile } = props;

    return (
        <div className="w-full h-full flex items-center justify-left">
            <img src={isMobile && !isFirst ? props['light-mobile'] : props['light']} className="w-full"/>
        </div>
    )
}

export default Logo;
