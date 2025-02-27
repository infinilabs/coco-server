import {
  Button,
  Cascader,
  Checkbox,
  DatePicker,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
  Switch,
  TreeSelect,
} from 'antd';
import {TypeList} from '@/components/datasource/type';
import {IndexingScope} from '@/components/datasource/indexing_scope';
import {DataSync} from '@/components/datasource/data_sync';

export function Component() {
  return <div className="bg-white pt-15px pb-15px">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>连接数据源</div>
          </div>
        </div>
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={{ size: 1 }}
            onValuesChange={()=>{}}
            colon={false}
          >
            <Form.Item label="数据源名称" name="name">
              <Input className='max-w-600px' placeholder='Please input datasource name'/>
            </Form.Item>
            <Form.Item label="数据源类型" name="type">
              <TypeList showTest={true}/>
            </Form.Item>
            <Form.Item label="索引范围">
              <IndexingScope/>
            </Form.Item>
            <Form.Item label="数据同步">
             <DataSync/>
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary'>保存</Button>
              <div className='mt-10px'>
                <Checkbox className='mr-5px'/>立即同步
              </div>
            </Form.Item>
          </Form>

        </div>
      </div>
  </div>
}