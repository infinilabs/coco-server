import { type FC } from "react";

import logoSvg from '../icons/logo-text-light.svg';
import logoDarkSvg from '../icons/logo-text-dark.svg';

import logoMobileSvg from "../icons/coco.svg"

interface LogoProps {
    isHome?: boolean;
    isMobile?: boolean;
    theme?: string;
    onLogoClick?: () => void;
    light?: string;
    light_mobile?: string;
    dark?: string;
    dark_mobile?: string;
}

export const Logo: FC<LogoProps> = (props) => {
    const { isHome, isMobile, theme, onLogoClick } = props;

    const logos = theme === 'dark' ? {
        logo: props.dark || logoDarkSvg,
        mobile: props.dark_mobile || logoMobileSvg,
    } : {
        logo: props.light || logoSvg,
        mobile: props.light_mobile || logoMobileSvg
    }

    return (
        <div className={`w-full h-full flex items-center justify-left max-w-inherit max-h-inherit object-contain`}>
            <img
                src={isMobile && !isHome ? logos.mobile : logos.logo}
                className={`${isMobile ? 'w-full h-full' : 'h-full w-auto'} cursor-pointer object-contain`}
                onClick={() => onLogoClick?.()}
                alt="Logo"
            />
        </div>
    )
}

export default Logo;
