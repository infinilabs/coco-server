import { Avatar, Button, Upload } from 'antd';

const MAX_IMAGE_BYTES = 1024 * 1024;

interface Props {
  label: string;
  value?: string;
  onChange: (value: string) => void;
  /** suggested image size shown under the controls, e.g. "≤ 320 × 56" */
  hint?: string;
  /** preview edge length in px */
  previewSize?: number;
}

/**
 * One branding-image slot: upload (local FileReader → base64 data URL, same
 * storage as the chat start-page logo), inline preview and reset.
 */
const ImageUploadItem = memo(({ hint, label, onChange, previewSize = 40, value }: Props) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);

  return (
    <div className="mb-16px">
      <div className="mb-8px settings-form-help">{label}</div>
      <div className="flex items-center" style={{ gap: 22 }}>
        {value ? (
          <Avatar
            shape="square"
            src={value}
            style={{ width: previewSize, height: previewSize, backgroundColor: 'var(--ant-color-fill-tertiary)' }}
          />
        ) : null}
        <Upload
          accept="image/*,.svg"
          action=""
          beforeUpload={(file) => {
            if (file.size > MAX_IMAGE_BYTES) {
              window.$message?.error(t('page.settings.appearance.labels.image_too_large'));
              return false;
            }
            setLoading(true);
            const reader = new FileReader();
            reader.readAsDataURL(file);
            reader.onload = () => {
              setLoading(false);
              onChange(reader.result as string);
            };
            return false;
          }}
          name="file"
          showUploadList={false}
        >
          <Button icon={<SvgIcon className="text-12px" icon="mdi:upload" />} loading={loading}>
            {t('common.upload')}
          </Button>
        </Upload>
        {value ? (
          <Button
            className="px-0"
            type="link"
            onClick={() => {
              setLoading(false);
              onChange('');
            }}
          >
            {t('common.reset')}
          </Button>
        ) : null}
      </div>
      {hint ? (
        <div className="mt-4px text-12px color-[var(--ant-color-text-description)]">{hint}</div>
      ) : null}
    </div>
  );
});

export default ImageUploadItem;
