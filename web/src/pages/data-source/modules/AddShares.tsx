import { Button, Form, Select, Space } from "antd";
import PrincipalSelect from "./PrincipalSelect";
import { addShares } from "@/service/api/share";

export default function AddShares(props) {

    const { hasCreate, permissionOptions = [], owner, editor, shares, onCancel, onSuccess, resource } = props;

    const { t } = useTranslation();
    const [form] = Form.useForm();
    const { defaultRequiredRule } = useFormRules();

    const onFinish = async (values) => {
        const { permission, shares = [] } = values;
        const formatShares = shares.map((item) => {
            const share = {
                ...(resource || {}),
                "principal_type": "user",
                "principal_id": item.id,
                permission,
            }
            return share
        })
        const res = await addShares({ shares: formatShares })
        if (res && !res.error) {
            window.$message?.success(t('common.addSuccess'));
            onSuccess && onSuccess()
        }
    }

    const excluded = useMemo(() => {
        const data = Array.isArray(shares) ? shares.map((item) => item.principal_id) : [];
        if (owner) {
            data.push(owner.id)
        }
        if (editor) {
            data.push(editor.id)
        }
        return data
    }, [owner, editor, shares])

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
                    <PrincipalSelect mode="multiple" excluded={excluded}/>
                </Form.Item>
                <Form.Item
                    label={t('page.datasource.labels.permission')}
                    name="permission"
                    rules={[defaultRequiredRule]}
                >
                    <Select options={permissionOptions.map((item) => ({ ...item, value: item.key}))}/>
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