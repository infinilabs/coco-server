import { useEffect, useState } from "react";

export default function Recommends(props) {
    const { onRecommend, showTitle = false } = props;
    const [data, setData] = useState([]);

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
                    <div className="text-16px text-[rgba(25,25,26,1)] mb-16px">相关搜索</div>
                )
            }
            {data.map((item, index) => (
                <div key={index}>
                    <div
                        className="bg-[rgba(241,241,241,1)] leading-[40px] h-[40px] mb-[8px] px-[12px] rounded-[12px] 
                            inline-block align-middle cursor-pointer"
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