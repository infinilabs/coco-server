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

  hostElement.setAttribute("data-fullscreen-host", "");

  const shadow = hostElement.attachShadow({ mode: "open" });

  const endpoint = options.endpoint || "$[[ENDPOINT]]";

  const loadIconfontScript = (src, svgKey) => new Promise((resolve, reject) => {
    const scriptElement = document.createElement("script");
    scriptElement.src = src;
    scriptElement.onload = () => {
      const svgElement = document.createElement("div");
      svgElement.style.height = "0";
      svgElement.style.overflow = "hidden";
      svgElement.innerHTML = window[svgKey] || "";
      shadow.appendChild(svgElement);
      resolve();
    };
    scriptElement.onerror = reject;
    shadow.appendChild(scriptElement);
  });

  const linkHref = endpoint + "/widgets/fullscreen/index.css?v=$[[VER]]";
  const linkElement = document.createElement("link");
  linkElement.rel = "stylesheet";
  linkElement.href = linkHref;
  shadow.appendChild(linkElement);

  linkElement.onload = async () => {
    await Promise.all([
      loadIconfontScript(`${endpoint}/assets/fonts/icons/iconfont.js`, "_iconfont_svg_string_4878526"),
      loadIconfontScript(`${endpoint}/assets/fonts/icons-app/iconfont.js`, "_iconfont_svg_string_4934333")
    ]);

    const wrapper = document.createElement("div");
    wrapper.classList.add("fullscreen-container");
    // ensure wrapper fills the host shadow container
    wrapper.style.width = "100%";
    wrapper.style.height = "100%";
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