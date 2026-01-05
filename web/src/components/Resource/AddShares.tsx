import { Button, Form, Select, Space } from "antd";
import PrincipalSelect from "./PrincipalSelect";

export interface AddSharesProps {
    hasCreate: boolean;
    permissionOptions?: any[];
    owner?: any;
    editor?: any;
    onCancel: () => void;
    resource?: any;
    currentShares?: any[];
    onSubmit: (data: any[]) => void;
    setLockOpen: (open: boolean) => void;
}

export default function AddShares(props: AddSharesProps) {

    const { hasCreate, permissionOptions = [], owner, editor, onCancel, resource, currentShares, onSubmit, setLockOpen } = props;

    const { t } = useTranslation();
    const [form] = Form.useForm();
    const { defaultRequiredRule } = useFormRules();

    const onFinish = async (values: any) => {
        const { permission, shares = [] } = values;
        onSubmit((currentShares ?? []).concat(shares.map((item: any) => {
            const share = {
                ...(resource || {}),
                "principal_type": item.type || "user",
                "principal_id": item.id,
                permission,
            }
            return share
        })))
    }

    const excluded = useMemo(() => {
        const data = Array.isArray(currentShares) ? currentShares.map((item) => item.principal_id) : [];
        if (owner) {
            data.push(owner.id)
        }
        if (editor) {
            data.push(editor.id)
        }
        return data
    }, [owner, editor, currentShares])

    if (!hasCreate) return null;

    return (
        <div>
            <Form
                colon={false}
                form={form}
                layout="vertical"
                onFinish={onFinish}
            >
                <Form.Item
                    label={t('page.datasource.labels.shareToPrincipal')}
                    name="shares"
                    rules={[defaultRequiredRule]}
                >
                    <PrincipalSelect mode="multiple" excluded={excluded} onDropdownVisibleChange={setLockOpen} />
                </Form.Item>
                <Form.Item
                    label={t('page.datasource.labels.permission')}
                    name="permission"
                    rules={[defaultRequiredRule]}
                >
                    <Select onDropdownVisibleChange={setLockOpen} options={permissionOptions.map((item: any) => ({ ...item, value: item.key}))}/>
                </Form.Item>
                <Form.Item className="mb-0px">
                    <div className="flex items-center justify-right">
                        <Space>
                            <Button className="w-80px" type="primary" ghost onClick={() => onCancel()}>
                                {t('common.cancel')}
                            </Button>
                            <Button className="w-80px" type="primary" htmlType="submit">
                                {t('common.ok')}
                            </Button>
                        </Space>
                    </div>
                </Form.Item>
            </Form>
        </div>
    )
}