export async function searchbox(options) {
  if (!options || !options.container) {
    console.error("Container is required in options.");
    return;
  }

  const hostElement = document.querySelector(options.container);
  if (!hostElement) {
    console.error("Container element not found:", options.container);
    return;
  }

  const shadow = hostElement.attachShadow({ mode: "open" });

  const endpoint = options.endpoint || "$[[ENDPOINT]]";
  const linkHref = endpoint + "/widgets/searchbox/index.css?v=$[[VER]]";

  const linkElement = document.createElement("link");
  linkElement.rel = "stylesheet";
  linkElement.href = linkHref;
  shadow.appendChild(linkElement);

  linkElement.onload = async () => {
    const wrapper = document.createElement("div");
    wrapper.classList.add("searchbox-container");
    shadow.appendChild(wrapper);

    const { searchbox: originalSearchbox } = await import(
      endpoint + "/widgets/searchbox/index.js?v=$[[VER]]"
    );

    const finalOptions = {
      ...options,
      container: wrapper,
      id: "$[[ID]]",
      server: endpoint,
      linkHref
    };

    originalSearchbox(finalOptions);
  };
}