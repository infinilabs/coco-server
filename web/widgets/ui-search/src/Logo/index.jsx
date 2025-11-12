import logoSvg from "../icons/logo.svg"
import logoMobileSvg from "../icons/coco.svg"

export function Logo(props) {
    const { isFirst, isMobile, onLogoClick } = props;

    return (
        <div className={`w-full h-full flex items-center justify-left ${isMobile ? '' : 'py-6px'}`}>
            <img
                src={isMobile && !isFirst ? (props['light-mobile'] || logoMobileSvg) : (props['light'] || logoSvg)}
                className="w-full cursor-pointer max-h-100%"
                onClick={onLogoClick}
                alt="Logo"
            />
        </div>
    )
}

export default Logo;
