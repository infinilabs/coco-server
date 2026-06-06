import { Dropdown, Slider, Space } from "antd";
import { ArrowUpDown, Calendar, ChevronDown, Crosshair, Heading, PanelLeftClose, RotateCw } from "lucide-react";
import { useTranslation } from 'react-i18next';
import { type FC } from "react";

export const Toolbar: FC = () => {
  const { t } = useTranslation();
  return (
    <div className="flex items-center w-full text-[#999] gap-16px">
      <PanelLeftClose className="w-16px h-16px text-16px"/>
      <Dropdown menu={{ items: [] }}>
          <Space size={4}>
            <Heading className="w-16px h-16px text-16px"/>
            {t('labels.semantic')}
            <ChevronDown className="w-16px h-16px text-16px"/>
          </Space>
      </Dropdown>
      <Space size={4}>
        <Crosshair className="w-16px h-16px text-16px"/>
        {t('labels.fuzziness')}
        <Slider className="w-75px" />
        <RotateCw className="w-16px h-16px text-16px"/>
      </Space>
      <Dropdown menu={{ items: [] }}>
          <Space size={4}>
            <ArrowUpDown className="w-16px h-16px text-16px"/>
            {t('labels.bestMatch')}
            <ChevronDown className="w-16px h-16px text-16px"/>
          </Space>
      </Dropdown>
      <Dropdown menu={{ items: [] }}>
          <Space size={4}>
            <Calendar className="w-16px h-16px text-16px"/>
            {t('labels.pastYear')}
            <ChevronDown className="w-16px h-16px text-16px"/>
          </Space>
      </Dropdown>
    </div>
  );
}

export default Toolbar;
