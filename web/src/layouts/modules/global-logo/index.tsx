import ClassNames from 'classnames';
import { Link } from 'react-router-dom';
import type { LinkProps } from 'react-router-dom';

import SystemLogo from '@/components/stateless/common/SystemLogo';
import DarkSystemLogo from '@/components/stateless/common/DarkSystemLogo';
import SystemLogoShort from '@/components/stateless/common/SystemLogoShort';
import { getAppearance } from '@/store/slice/server';

interface Props extends Omit<LinkProps, 'to'> {
  /** Render the dark-mode logo variant */
  darkMode?: boolean;
  /** Whether to show the title */
  showTitle?: boolean;
  siderCollapse?: boolean;
}
const GlobalLogo: FC<Props> = memo(
  ({ className, darkMode = false, showTitle = true, siderCollapse = false, ...props }) => {
    const { t } = useTranslation();

    const appearance = useAppSelector(getAppearance);
    const configured = appearance?.logo;
    const siteTitle = appearance?.title;

    const renderIcon = () =>
      configured?.icon ? (
        <img className="h-38px w-38px object-contain" src={configured.icon} />
      ) : (
        <SystemLogoShort />
      );

    const renderLogo = () => {
      if (siderCollapse) {
        return renderIcon();
      }
      // with a configured site title the header is "icon + title": the bundled
      // wordmark already carries "Coco Server" text, so stacking the title
      // next to it reads as two brand names
      if (siteTitle) {
        return renderIcon();
      }
      if (darkMode) {
        return configured?.dark ? (
          <div className="h-full w-full px-24px">
            <img className="h-full w-full object-contain" src={configured.dark} />
          </div>
        ) : (
          <div className="h-full w-full px-24px">
            <DarkSystemLogo />
          </div>
        );
      }
      return configured?.light ? (
        <img className="h-55px max-w-full px-24px object-contain" src={configured.light} />
      ) : (
        <SystemLogo className="h-55px px-24px text-32px text-primary" />
      );
    };

    return (
      <Link
        className={ClassNames('w-full flex-center nowrap-hidden', className)}
        to="/"
        {...props}
      >
        {renderLogo()}
        {showTitle && siteTitle ? (
          <h2 className="pl-12px truncate text-17px text-primary font-bold transition duration-300 ease-in-out">
            {siteTitle}
          </h2>
        ) : null}
      </Link>
    );
  }
);

export default GlobalLogo;
