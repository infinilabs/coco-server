import ClassNames from 'classnames';

import SystemLogo from './SystemLogo';

const loadingClasses = [
  'left-0 top-0',
  'left-0 bottom-0 animate-delay-500',
  'right-0 top-0 animate-delay-1000',
  'right-0 bottom-0 animate-delay-1500'
];

const GlobalLoading = memo(() => {
  const { t } = useTranslation();

  return (
    <div className="fixed-center flex-col">
      <SystemLogo className="w-320px h-128px text-primary" />
      <div className="w-48px h-48px my-24px">
        <div className="relative h-full animate-spin">
          {loadingClasses.map(item => {
            return (
              <div
                className={ClassNames('absolute w-16px h-16px bg-primary rounded-8px animate-pulse ', item)}
                key={item}
              />
            );
          })}
        </div>
      </div>
      {/* <h2 className="text-28px text-#646464 font-500">{t('system.title')}</h2> */}
    </div>
  );
});

export default GlobalLoading;
