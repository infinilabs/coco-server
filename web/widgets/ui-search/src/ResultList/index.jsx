import NormalList from "./NormalList";
import ImageList from "./ImageList";

export const LIST_TYPES = [
  {
    type: "all",
    component: NormalList,
    showAIOverview: true,
  },
  {
    type: "doc",
    component: NormalList,
    showAIOverview: false,
  },
  {
    type: "image",
    component: ImageList,
    showAIOverview: false,
  },
];
