import { Dropdown, Slider, Space } from "antd";
import { ArrowUpDown, Calendar, ChevronDown, Crosshair, Heading, PanelLeftClose, RotateCw } from "lucide-react";

export function Toolbar(props) {
  const {  } = props;
  return (
    <div className="flex items-center w-full text-[#999] gap-16px">
      <PanelLeftClose className="w-16px h-16px text-16px"/>
      <Dropdown menu={{ items: [] }}>
          <Space size={4}>
            <Heading className="w-16px h-16px text-16px"/>
            语义搜索
            <ChevronDown className="w-16px h-16px text-16px"/>
          </Space>
      </Dropdown>
      <Space size={4}>
        <Crosshair className="w-16px h-16px text-16px"/>
        模糊程度
        <Slider className="w-75px" />
        <RotateCw className="w-16px h-16px text-16px"/>
      </Space>
      <Dropdown menu={{ items: [] }}>
          <Space size={4}>
            <ArrowUpDown className="w-16px h-16px text-16px"/>
            最佳匹配
            <ChevronDown className="w-16px h-16px text-16px"/>
          </Space>
      </Dropdown>
      <Dropdown menu={{ items: [] }}>
          <Space size={4}>
            <Calendar className="w-16px h-16px text-16px"/>
            最近一年
            <ChevronDown className="w-16px h-16px text-16px"/>
          </Space>
      </Dropdown>
    </div>
  );
}

export default Toolbar;
