import { Form } from "antd";
import {MultiURLInput} from '@/components/datasource/type/urls';
export default () => {
  const { t } = useTranslation();
  return  (<>
     <Form.Item label={t('page.datasource.new.labels.site_urls')} name="urls" rules={[{ required: true, message: 'Please input site url!' }]} >
       <MultiURLInput showLabel={false}/>
     </Form.Item>
 </>)
}