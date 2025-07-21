import {
  Button,
  Form,
  Input,
  message,
  Select,
  Switch,
  InputNumber,
  Space,
  Spin,
} from "antd";
import type { FormProps } from "antd";
import {
  DeleteOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { useRequest, useLoading } from "@sa/hooks";

import { getConnectorIcons } from "@/service/api/connector";
import { IconSelector } from "../../connector/new/icon_selector";
import {
  fetchDataSourceList,
  getEnabledModelProviders,
} from "@/service/api";
import { searchMCPServer } from "@/service/api/mcp-server";
import { AssistantMode } from "./AssistantMode";
import { DatasourceConfig } from "./DatasourceConfig";
import { MCPConfig } from "./MCPConfig";
import { DeepThink } from "./DeepThink";
import { formatESSearchResult } from "@/service/request/es";
import ModelSelect from "./ModelSelect";
import { ToolsConfig } from "./ToolsConfig";
import { getUUID } from "@/utils/common";
import { Tags } from '@/components/common/tags';
import { getAssistantCategory } from "@/service/api/assistant";
import { UploadConfig } from "./UploadConfig";

interface AssistantFormProps {
  initialValues: any;
  onSubmit: (
    values: any,
    startLoading: () => void,
    endLoading: () => void,
  ) => void;
  mode: string;
  loading: boolean;
}

export const EditForm = memo((props: AssistantFormProps) => {
  const { initialValues = {}, onSubmit, mode } = props;
  const [form] = Form.useForm();
  useEffect(() => {
    if (initialValues) {
      if (initialValues.datasource?.filter) {
        initialValues.datasource.filter = JSON.stringify(
          initialValues.datasource.filter,
        );
      }
      form.setFieldsValue({
        ...initialValues,
        icon: initialValues.icon || 'font_coco'
      });
    }
  }, [initialValues]);
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();

  const onFinish: FormProps<any>["onFinish"] = (values) => {
    if (values.datasource?.filter) {
      try {
        values.datasource.filter = JSON.parse(values.datasource.filter);
      } catch (e) {
        message.error("Datasource filter is not valid JSON");
        return;
      }
    } else {
      if (!values.datasource) values.datasource = {}
      values.datasource.filter = null;
    }
    if (values.upload?.allowed_file_extensions) {
      values.upload.allowed_file_extensions = values.upload.allowed_file_extensions.filter((item) => !!item)
    }
    onSubmit?.({
      ...values,
      category: values?.category?.[0] || '',
    }, startLoading, endLoading);
  };

  const onFinishFailed: FormProps<any>["onFinishFailed"] = (errorInfo) => {
    console.log("Failed:", errorInfo);
  };
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then((res) => {
      if (res.data?.length > 0) {
        setIconsMeta(res.data);
      }
    });
  }, []);
  const { defaultRequiredRule, formRules } = useFormRules();

  const [showAdvanced, setShowAdvanced] = useState(false);
  const {
    data: result,
    run,
    loading: dataSourceLoading,
  } = useRequest(fetchDataSourceList, {
    manual: true,
  });

  useEffect(() => {
    run({
      from: 0,
      size: 10000,
    });
  }, []);

  const dataSource = useMemo(() => {
    return result?.hits?.hits?.map((item) => ({ ...item._source })) || [];
  }, [JSON.stringify(result)]);

  const { data: modelsResult, run: fetchModelProviders } = useRequest(
    getEnabledModelProviders,
    {
      manual: true,
    },
  );
  const modelProviders = useMemo(() => {
    if (!modelsResult) return [];
    const res = formatESSearchResult(modelsResult);
    return res.data;
  }, [JSON.stringify(modelsResult)]);
  useEffect(() => {
    fetchModelProviders(10000);
  }, []);

  const { data: mcpServerResult, run: fetchMCPServers } = useRequest(
    searchMCPServer,
    {
      manual: true,
    },
  );

  useEffect(() => {
    fetchMCPServers({
      from: 0,
      size: 10000,
    });
  }, []);

  const mcpServers = useMemo(() => {
    return (
      mcpServerResult?.hits?.hits?.map((item) => ({ ...item._source })) || []
    );
  }, [JSON.stringify(mcpServerResult)]);

  const [assistantMode, setAssistantMode] = useState(
    initialValues?.mode || "simple",
  );
  useEffect(() => {
    if (initialValues?.type) {
      setAssistantMode(initialValues.type);
    }
  }, [initialValues?.type]);
  const handleAssistantModeChange = (value: string) => {
    setAssistantMode(value);
  };

  const [suggestedChatChecked, setSuggestedChatChecked] = useState(
    initialValues?.chat_settings?.suggested?.enabled || false,
  );
  useEffect(() => {
    setSuggestedChatChecked(
      initialValues?.chat_settings?.suggested?.enabled || false,
    );
  }, [initialValues?.chat_settings?.suggested?.enabled]);

  const [categories, setCategories] = useState([]);
  useEffect(() => {
    getAssistantCategory().then(({ data }) => {
      if (!data?.error) {
        const newData = formatESSearchResult(data);
        const cates = newData.aggregations.categories.buckets.map((item: any) => {
          return item.key;
        });
        setCategories(cates);
      }
    });
  }, []);

  const commonFormItemsClassName = `${showAdvanced || assistantMode === "deep_think" ? "" : "h-0px m-0px overflow-hidden"}`

  const commonFormItems = (
    <>
      <Form.Item
        label={t("page.assistant.labels.datasource")}
        rules={[{ required: true }]}
        name={["datasource"]}
        className={commonFormItemsClassName}
      >
        <DatasourceConfig
          options={[{ label: "*", value: "*" }].concat(
            dataSource.map((item) => ({
              label: item.name,
              value: item.id,
            })),
          )}
        />
      </Form.Item>
      <Form.Item
        label={t("page.assistant.labels.mcp_servers")}
        rules={[{ required: true }]}
        name="mcp_servers"
        className={commonFormItemsClassName}
      >
        <MCPConfig
          modelProviders={modelProviders}
          options={[{ label: "*", value: "*" }].concat(
            mcpServers.map((item) => ({
              label: item.name,
              value: item.id,
            })),
          )}
        />
      </Form.Item>
      <Form.Item
        label={t("page.assistant.labels.upload")}
        rules={[{ required: true }]}
        name="upload"
        className={commonFormItemsClassName}
      >
        <UploadConfig />
      </Form.Item>
      <Form.Item label={t("page.assistant.labels.tools")} name="tools" className={commonFormItemsClassName}>
        <ToolsConfig />
      </Form.Item>
      <Form.Item
        name={"keepalive"}
        label={t("page.assistant.labels.keepalive")}
        rules={[defaultRequiredRule]}
        className={commonFormItemsClassName}
      >
        <Input className="max-w-600px" />
      </Form.Item>
    </>
  )

  return (
    <Spin spinning={props.loading || loading || false}>
      <Form
        labelCol={{ span: 4 }}
        wrapperCol={{ span: 18 }}
        layout="horizontal"
        initialValues={initialValues}
        colon={false}
        form={form}
        autoComplete="off"
        onFinish={onFinish}
        onFinishFailed={onFinishFailed}
      >
        <Form.Item
          label={t("page.assistant.labels.name")}
          rules={[{ required: true }]}
          name="name"
        >
          <Input className="max-w-600px" />
        </Form.Item>
        <Form.Item
          label={t("page.assistant.labels.description")}
          name="description"
        >
          <Input className="max-w-600px" />
        </Form.Item>
        <Form.Item
          label={t("page.assistant.labels.icon")}
          name="icon"
          rules={[{ required: true }]}
        >
          <IconSelector
            type="connector"
            icons={iconsMeta}
            className="max-w-600px"
          />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.category')}
          name="category"
        >
          <Select
            className="max-w-600px"
            maxCount={1}
            mode="tags"
            placeholder="Select or input a category"
            options={categories.map(cate => {
              return { value: cate };
            })}
          />
        </Form.Item>
        <Form.Item
          label={t('page.assistant.labels.tags')}
          name="tags"
        >
          <Tags />
        </Form.Item>
        <Form.Item
          label={t("page.assistant.labels.type")}
          name="type"
          rules={[{ required: true }]}
        >
          <AssistantMode onChange={handleAssistantModeChange} />
        </Form.Item>
        <Form.Item
          label={t("page.assistant.labels.answering_model")}
          rules={[{ required: true }]}
          name={["answering_model"]}
        >
          <ModelSelect width="600px" modelType="answering_model" providers={modelProviders} />
        </Form.Item>
        {assistantMode === "deep_think" && (
          <Form.Item label={t("page.assistant.labels.deep_think_model")}>
            <DeepThink className="max-w-600px" providers={modelProviders} />
          </Form.Item>
        )}
        { assistantMode === "deep_think" ? commonFormItems : null }
        <Form.Item
          name="role_prompt"
          label={t("page.assistant.labels.role_prompt")}
        >
          <Input.TextArea
            placeholder="Please enter the role prompt instructions"
            style={{ height: 320 }}
            className="w-600px"
          />
        </Form.Item>
        <Form.Item label={t("page.assistant.labels.greeting_settings")} name={["chat_settings", "greeting_message"]}>
          <Input.TextArea className="w-600px" />
        </Form.Item>
        <Form.Item label={t("page.assistant.labels.enabled")} name="enabled">
          <Switch size="small" />
        </Form.Item>
        <Form.Item label=" ">
          <Button
            type="link"
            className="p-0"
            onClick={() => setShowAdvanced(!showAdvanced)}
          >
            {t("common.advanced")}{" "}
            <SvgIcon
              icon={`${showAdvanced ? "mdi:chevron-up" : "mdi:chevron-down"}`}
            />
          </Button>
        </Form.Item>
        { assistantMode === "simple" ? commonFormItems : null }
        <Form.Item
          className={`${showAdvanced ? "" : "h-0px m-0px overflow-hidden"}`}
          label={t("page.assistant.labels.chat_settings")}
        >
          <div className="max-w-600px">
            <SuggestedChatForm checked={suggestedChatChecked} />
            {/* <div>
              <p>{t("page.assistant.labels.input_preprocessing")}</p>
              <div className="text-gray-400 leading-6 mb-1">
                {t("page.assistant.labels.input_preprocessing_desc")}
              </div>
              <Form.Item name={["chat_settings", "input_preprocess_tpl"]}>
                <Input.TextArea
                  placeholder={t(
                    "page.assistant.labels.input_preprocessing_placeholder",
                  )}
                  className="w-600px"
                />
              </Form.Item>
            </div> */}
            <div>
              <p className="mb-1">{t("page.assistant.labels.input_placeholder")}</p>
              <Form.Item name={["chat_settings", "placeholder"]}>
                <Input.TextArea
                  className="w-600px"
                />
              </Form.Item>
            </div>
            <div className="flex justify-between items-center">
              <div>
                <p>{t("page.assistant.labels.history_message_number")}</p>
                <div className="text-gray-400 leading-6 mb-1">
                  {t("page.assistant.labels.history_message_number_desc")}
                </div>
              </div>
              <Form.Item name={["chat_settings", "history_message", "number"]}>
                <InputNumber min={0} max={64} />
              </Form.Item>
            </div>
            <div className="flex justify-between items-center">
              <div>
                <p>
                  {t(
                    "page.assistant.labels.history_message_compression_threshold",
                  )}
                </p>
                <div className="text-gray-400 leading-6 mb-1">
                  {t(
                    "page.assistant.labels.history_message_compression_threshold_desc",
                  )}
                </div>
              </div>
              <Form.Item
                name={[
                  "chat_settings",
                  "history_message",
                  "compression_threshold",
                ]}
              >
                <InputNumber min={500} max={4000} />
              </Form.Item>
            </div>
            <div className="flex justify-between items-center">
              <div>
                <p>{t("page.assistant.labels.history_summary")}</p>
                <div className="text-gray-400 leading-6 mb-1">
                  {t("page.assistant.labels.history_summary_desc")}
                </div>
              </div>
              <Form.Item name={["chat_settings", "history_message", "summary"]}>
                <Switch size="small" />
              </Form.Item>
            </div>
          </div>
        </Form.Item>
        <Form.Item label=" ">
          <Button type="primary" htmlType="submit">
            {t("common.save")}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});

export const SuggestedChatForm = ({ checked }: { checked: boolean }) => {
  const { t } = useTranslation();
  const [enabled, setEnabled] = useState(checked);
  useEffect(() => {
    setEnabled(checked);
  }, [checked]);
  const onEnabledChange = (checked: boolean) => {
    setEnabled(checked);
  };
  return (
    <div>
      <div className="text-gray-400 leading-6 mb-1 flex gap-1 items-center">
        {t("page.assistant.labels.suggested_chat")}{" "}
        <Form.Item
          name={["chat_settings", "suggested", "enabled"]}
          style={{ margin: 0 }}
        >
          <Switch size="small" onChange={onEnabledChange} defaultChecked />
        </Form.Item>
      </div>
      <Form.Item
        name={["chat_settings", "suggested", "questions"]}
        className={`${enabled ? "" : "h-0px m-0px overflow-hidden"}`}
      >
        <SuggestedChat />
      </Form.Item>
    </div>
  );
};

export const SuggestedChat = ({ value = [], onChange }: any) => {
  const initialValue = useMemo(() => {
    const iv = (value || []).map((v: string) => ({
      value: v,
      key: getUUID(),
    }));
    return iv.length ? iv : [{ value: "", key: getUUID() }];
  }, [value]);

  const [innerValue, setInnerValue] =
    useState<{ value: string; key: string }[]>(initialValue);
  const prevValueRef = useRef<string[]>([]);

  // Prevent unnecessary updates
  useEffect(() => {
    if (JSON.stringify(prevValueRef.current) !== JSON.stringify(value)) {
      prevValueRef.current = value;
      const iv = (value || []).map((v: string) => ({
        value: v,
        key: getUUID(),
      }));
      setInnerValue(iv.length ? iv : [{ value: "", key: getUUID() }]);
    }
  }, [value]);

  const onDeleteClick = (key: string) => {
    const newValues = innerValue.filter((v) => v.key !== key);
    setInnerValue(
      newValues.length ? newValues : [{ value: "", key: getUUID() }],
    );
    const newValue = newValues.map((v) => v.value);
    prevValueRef.current = newValue;
    onChange?.(newValue);
  };

  const onAddClick = () => {
    setInnerValue([...innerValue, { value: "", key: getUUID() }]);
  };

  const onItemChange = (key: string, newValue: string) => {
    const updatedValues = innerValue.map((v) =>
      v.key === key ? { ...v, value: newValue } : v,
    );
    setInnerValue(updatedValues);
    const filterValues = updatedValues
      .filter((v) => v.value != "")
      .map((v) => v.value);
    prevValueRef.current = filterValues;
    onChange?.(filterValues);
  };

  const { t } = useTranslation();

  return (
    <div>
      {innerValue.map((v) => (
        <div key={v.key} className="flex items-center mb-15px">
          <Input
            value={v.value}
            placeholder="eg: what is easysearch?"
            onChange={(e) => {
              onItemChange(v.key, e.target.value);
            }}
          />
          <div
            className="cursor-pointer ml-15px"
            onClick={() => onDeleteClick(v.key)}
          >
            <DeleteOutlined />
          </div>
        </div>
      ))}
      <Button
        type="primary"
        icon={<PlusOutlined />}
        style={{ width: 80 }}
        onClick={onAddClick}
      ></Button>
    </div>
  );
};