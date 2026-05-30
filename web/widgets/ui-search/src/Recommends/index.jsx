import { useEffect, useState } from "react";
import { useTranslation } from 'react-i18next';

export default function Recommends(props) {
    const { onRecommend, showTitle = false } = props;
    const [data, setData] = useState([]);
    const { t } = useTranslation();

    useEffect(() => {
        onRecommend?.(res => {
            if (Array.isArray(res?.recommendations)) {
                setData(res?.recommendations.sort((a, b) => b.score - a.score));
            }
        });
    }, []);

    return (
        <div className="recommends-container">
            {
                showTitle && (
                    <div className="text-16px text-[rgba(25,25,26,1)] dark:text-[rgba(230,230,230,1)] mb-16px">{t('labels.relatedSearch')}</div>
                )
            }
            {data.map((item, index) => (
                <div key={index}>
                    <div
                        className="bg-[rgba(241,241,241,1)] dark:bg-[rgba(55,55,55,1)] text-[rgba(25,25,26,1)] dark:text-[rgba(230,230,230,1)] leading-[40px] h-[40px] mb-[8px] px-[12px] rounded-[12px] 
                            inline-block align-middle cursor-pointer max-w-full overflow-hidden text-ellipsis whitespace-nowrap"
                        onClick={() => {
                            if (item.url?.startsWith('http')) {
                                window.open(item.url)
                            }
                        }}
                    >
                        {item.title}
                    </div>
                </div>
            ))}
        </div>
    );
}