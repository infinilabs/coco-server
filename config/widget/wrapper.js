import { searchbox as originalSearchbox } from "$[[ENDPOINT]]/widgets/searchbox/index.js?v=$[[VER]]";

export function searchbox(options) {
  if (!options || !options.container) {
    console.error("Container is required in options.");
    return;
  }

  const hostElement = document.querySelector(options.container);
  if (!hostElement) {
    console.error("Container element not found:", options.container);
    return;
  }

  // Attach Shadow DOM
  const shadow = hostElement.attachShadow({ mode: "open" });

  // Load external CSS into Shadow DOM
  const linkHref = "$[[ENDPOINT]]/widgets/searchbox/index.css?v=$[[VER]]"
  const linkElement = document.createElement("link");
  linkElement.rel = "stylesheet";
  linkElement.href = linkHref;
  shadow.appendChild(linkElement);

  // Create wrapper div inside Shadow DOM
  const wrapper = document.createElement("div");
  wrapper.classList.add("searchbox-container");
  shadow.appendChild(wrapper);

  // Set default server but keep other options flexible
  const finalOptions = {
    ...options,
    container: wrapper,
    id: "$[[ID]]",
    server: "$[[ENDPOINT]]",
    token: "$[[TOKEN]]",
    linkHref
  };

  // Call the original searchbox function
  originalSearchbox(finalOptions);
}