export async function fullscreen(options) {
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

  const linkHref = endpoint + "/widgets/fullscreen/index.css?v=$[[VER]]";
  const linkElement = document.createElement("link");
  linkElement.rel = "stylesheet";
  linkElement.href = linkHref;
  shadow.appendChild(linkElement);

  linkElement.onload = async () => {
    const wrapper = document.createElement("div");
    wrapper.classList.add("fullscreen-container");
    shadow.appendChild(wrapper);

    const { fullscreen: originalFullscreen } = await import(
      endpoint + "/widgets/fullscreen/index.js?v=$[[VER]]"
    );

    const finalOptions = {
      ...options,
      shadow,
      container: wrapper,
      id: "$[[ID]]",
      server: endpoint,
      linkHref
    };

    originalFullscreen(finalOptions);
  };
}