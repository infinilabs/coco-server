export function setupMicro(root, renderRoot) {

    window.__WUJIE_MOUNT = () => {
      renderRoot(root)
    };
    window.__WUJIE_UNMOUNT = () => {
      root.unmount()
    };
    window.__WUJIE.mount()
    
    setupIcons();
    setupHistoryHooks();
}

function setupIcons() {
    const parentDoc = window.$wujie?.props?.parentDocument;
    if (!parentDoc) return; 

    const existingMicroIcon = parentDoc.getElementById('__MICRO__SVG_ICON_LOCAL__');
    if (existingMicroIcon) return;

    const sourceIcon = document.getElementById('__SVG_ICON_LOCAL__');
    if (!sourceIcon) return;

    const clonedIcon = sourceIcon.cloneNode(true) as any;
    clonedIcon.id = '__MICRO__SVG_ICON_LOCAL__';

    parentDoc.body.appendChild(clonedIcon);
}

function setupHistoryHooks() {
    let lastUrl = window.location.href;

    function handleUrlChange() {
        const currentUrl = window.location.href;
        if (currentUrl !== lastUrl) {
            lastUrl = currentUrl;
            window.$wujie?.props?.onRouteChange({ 
                url: currentUrl, 
            })
        }
    }

    const originalPushState = window.history.pushState;
    window.history.pushState = function(...args) {
        originalPushState.apply(this, args);
        handleUrlChange(); 
    };

    const originalReplaceState = window.history.replaceState;
    window.history.replaceState = function(...args) {
        originalReplaceState.apply(this, args);
        handleUrlChange(); 
    };
}