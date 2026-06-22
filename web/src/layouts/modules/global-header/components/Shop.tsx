import { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import { getProviderInfo } from '@/store/slice/server';
import { ShoppingBag } from 'lucide-react';

const isValidUrl = (value: any) => {
  if (!value || typeof value !== 'string') return false;
  try {
    // eslint-disable-next-line no-new
    new URL(value);
    return true;
  } catch (e) {
    return false;
  }
}

export const isStoreEnabled = (providerInfo: any) => {
  const endpoint = providerInfo?.store?.endpoint;
  return providerInfo?.store?.enabled && isValidUrl(endpoint);
}

const Shop = memo(() => {
  const { t } = useTranslation();
  const integratedStoreModalRef = useRef<IntegratedStoreModalRef>(null);
  const providerInfo = useAppSelector(getProviderInfo);

  if (!isStoreEnabled(providerInfo)) return null;

  return (
    <>
      <ButtonIcon
        className="px-12px"
        tooltipContent={t('page.integratedStoreModal.title')}
        onClick={() => {
          integratedStoreModalRef.current?.open('ai-assistant');
        }}
      >
        <ShoppingBag className="size-4" />
      </ButtonIcon>
      <IntegratedStoreModal ref={integratedStoreModalRef} />
    </>
  );
});

export default Shop;
