
export function Logo(props) {
    const { isFirst, isMobile, onLogoClick } = props;

    return (
        <div className="w-full h-full flex items-center justify-left">
            <img
                src={isMobile && !isFirst ? props['light-mobile'] : props['light']}
                className="w-full cursor-pointer"
                onClick={onLogoClick}
                alt="Logo"
            />
        </div>
    )
}

export default Logo;
