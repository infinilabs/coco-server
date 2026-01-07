import { memo } from "react";
import { Masonry } from "antd";
import { ChevronRight } from "lucide-react";

export function ImageList(props) {
  const { getDetailContainer, data = [], isMobile, loading, hasMore } = props;

  const items = Array.from({ length: 50 }, (_, index) => {
    const key = index + 1;
    const width = Math.round(Math.random() * (600 - 200) + 200);
    const height = Math.round(Math.random() * (600 - 200) + 200);
    const thumbnailWidth = width * 0.5;
    const thumbnailHeight = height * 0.5;

    return {
      id: key,
      title: `title${key}`,
      category: "Category",
      thumbnail: `https://picsum.photos/${thumbnailWidth}/${thumbnailHeight}`,
      source: {
        name: "Source",
      },
      metadata: {
        image_media_metadata: {
          width,
          height,
        },
      },
    };
  });

  return (
    <Masonry
      columns={{
        xl: 6,
        lg: 5,
        md: 4,
        sm: 3,
        xs: 2,
      }}
      gutter={16}
      items={items.map((item) => ({
        key: item.id,
        data: item,
      }))}
      itemRender={({ data }) => (
        <div className="group relative cursor-pointer">
          <div className="w-full rounded-lg overflow-hidden">
            <img
              src={data?.thumbnail}
              alt={data?.title}
              className="w-full object-cover"
            />
          </div>

          <div className="absolute left-0 bottom-0 w-full p-3 text-3.5 text-white opacity-0 transition group-hover:opacity-100">
            <div>{data?.title}</div>

            <div className="inline-flex items-center flex-wrap gap-0.5">
              <span>{data?.source?.name}</span>
              <ChevronRight className="size-3" />
              <span>{data?.category}</span>
            </div>
          </div>
        </div>
      )}
    />
  );
}

export default memo(ImageList);
