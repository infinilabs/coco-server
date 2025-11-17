import logoSvg from "../icons/logo.svg"
import logoDarkSvg from "../icons/logo-dark.svg"
import logoMobileSvg from "../icons/coco.svg"

export function Logo(props) {
    const { isFirst, isMobile, theme, onLogoClick } = props;

    const logos = theme === 'dark' ? {
        logo: props['dark'] || logoDarkSvg,
        mobile: props['dark_mobile'] || logoMobileSvg,
    } : {
        logo: props['light'] || logoSvg,
        mobile: props['light_mobile'] || logoMobileSvg
    }

    return (
        <div className={`w-full h-full flex items-center justify-left ${isMobile ? '' : 'py-8px'}`}>
            <img
                src={isMobile && !isFirst ? logos.mobile : logos.logo}
                className="w-full cursor-pointer max-h-100%"
                onClick={onLogoClick}
                alt="Logo"
            />
        </div>
    )
}

export default Logo;
