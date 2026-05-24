import { getDefaultModelTips, setDefaultModelTips } from "@/store/slice/server";
import { localStg } from "@/utils/storage";
import { CloseOutlined } from "@ant-design/icons";
import { Typography } from "antd";

const TopBanner = memo(() => {

  const nav = useNavigate();
  const { t } = useTranslation();
  
  const defaultModelTips = useAppSelector(getDefaultModelTips);
  const dispatch = useAppDispatch();

  if (!defaultModelTips) {
    return null;
  }

  return (
    <div className="h-48px bg-[var(--ant-color-primary)] w-full color-white flex gap-8px px-16px">
      <div className="flex-1 flex items-center justify-center gap-8px min-w-0">
        <Typography.Text className="!color-white truncate" ellipsis={{ tooltip: true }}>
          {t('page.guide.labels.tips')}
        </Typography.Text>
        <Typography.Link className="!color-white !underline flex-shrink-0">
          {t('page.guide.labels.tipsSettings')}
        </Typography.Link>
      </div>
      <div className="flex items-center justify-center gap-16px flex-shrink-0 min-w-[140px]">
        <Typography.Link className="!color-white flex-shrink-0" onClick={() => {
          dispatch(setDefaultModelTips(false));
          localStg.set('ignoreDefaultModelTips', 'true');
        }}>
          [{t('page.guide.labels.ignoreTips')}]
        </Typography.Link>
        <CloseOutlined className="cursor-pointer flex-shrink-0" onClick={() => { 
          dispatch(setDefaultModelTips(false));
         }} />
      </div>
    </div>
  );
});

export const TopBannerHeight = 48;

export default TopBanner;
