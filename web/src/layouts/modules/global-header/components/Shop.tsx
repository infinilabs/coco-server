import { IntegratedStoreModalRef } from '@/components/common/IntegratedStoreModal';
import { ShoppingBag } from 'lucide-react';

const Shop = memo(() => {
  const { t } = useTranslation();
  const integratedStoreModalRef = useRef<IntegratedStoreModalRef>(null);

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
