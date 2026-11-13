import { type CSSProperties, type FC } from "react";

interface WelcomeProps {
    text?: string;
    isMobile?: boolean;
    /** font size in px (default 30, 20 on mobile) */
    fontSize?: number;
    /** brand-gradient text; off renders solid base-text color */
    gradient?: boolean;
}

export const Welcome: FC<WelcomeProps> = (props) => {
    const { text, isMobile, fontSize, gradient = true } = props;

    if (!text) return null

    // hero treatment: the welcome/slogan is the home view's headline — large,
    // semibold; gradient mode clips a brand-primary gradient into the glyphs
    const size = fontSize || (isMobile ? 20 : 30);
    const style: CSSProperties = gradient
        ? {
            backgroundImage: 'linear-gradient(90deg, rgb(var(--ui-search--primary-color)), rgb(var(--ui-search--primary-300-color)))',
            WebkitBackgroundClip: 'text',
            backgroundClip: 'text',
            color: 'transparent'
        }
        : { color: 'rgb(var(--ui-search--base-text-color))' };

    return (
        <div
            className={`w-full text-center font-600 tracking-wide ${isMobile ? 'leading-[32px]' : 'leading-[46px]'}`}
            style={{ fontSize: `${size}px`, ...style }}
        >
            {text}
        </div>
    )
}

export default Welcome;
