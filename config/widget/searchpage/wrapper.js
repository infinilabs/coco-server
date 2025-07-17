export async function searchPage(options) {
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

  const linkHref = "$[[ENDPOINT]]/widgets/searchpage/index.css?v=$[[VER]]";
  const linkElement = document.createElement("link");
  linkElement.rel = "stylesheet";
  linkElement.href = linkHref;
  shadow.appendChild(linkElement);

  linkElement.onload = async () => {
    const wrapper = document.createElement("div");
    wrapper.classList.add("searchpage-container");
    shadow.appendChild(wrapper);

    const { searchPage: originalSearchPage } = await import(
      "$[[ENDPOINT]]/widgets/searchpage/index.js?v=$[[VER]]"
    );

    const finalOptions = {
      ...options,
      container: wrapper,
      id: "$[[ID]]",
      server: "$[[ENDPOINT]]",
      token: "$[[TOKEN]]",
      linkHref
    };

    originalSearchPage(finalOptions);
  };
}