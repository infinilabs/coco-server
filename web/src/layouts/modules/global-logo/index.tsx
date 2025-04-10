import ClassNames from 'classnames';
import { Link } from 'react-router-dom';
import type { LinkProps } from 'react-router-dom';

import SystemLogo from '@/components/stateless/common/SystemLogo';

interface Props extends Omit<LinkProps, 'to'> {
  /** Whether to show the title */
  showTitle?: boolean;
  siderCollapse?: boolean;
}
const GlobalLogo: FC<Props> = memo(
  ({ className, darkMode = false, showTitle = true, siderCollapse = false, ...props }) => {
    const { t } = useTranslation();

    return (
      <Link
        className={ClassNames('w-full flex-center nowrap-hidden', className)}
        to="/"
        {...props}
      >
        {siderCollapse ? (
          <SystemLogoShort />
        ) : darkMode ? (
          <div className="h-full w-full px-24px">
            <DarkSystemLogo />
          </div>
        ) : (
          <SystemLogo className="h-55px px-24px text-32px text-primary" />
        )}
        <h2
          className="pl-8px text-16px text-primary font-bold transition duration-300 ease-in-out"
          style={{ display: showTitle ? 'block' : 'none' }}
        >
          {t('system.title')}
        </h2>
      </Link>
    );
  }
);

export default GlobalLogo;
