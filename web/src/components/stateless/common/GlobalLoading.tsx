import ClassNames from 'classnames';

import loadingIcon from '@/assets/svg-icon/file-loading.svg';

// hold the icon back briefly so fast loads never flash it on screen
const ICON_DELAY_MS = 300;

const GlobalLoading = memo((props: any) => {
  const { className } = props;
  const [showIcon, setShowIcon] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setShowIcon(true), ICON_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <div className={ClassNames('fixed-center bg-[rgb(var(--layout-bg-color))]', className)}>
      {showIcon && <img className="h-48px w-48px" src={loadingIcon} alt="" />}
    </div>
  );
});

export default GlobalLoading;
