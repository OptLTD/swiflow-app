document.addEventListener('DOMContentLoaded', function() {
    const style = document.createElement('style');
    style.innerHTML = `
        /* tauri 部分处理 start */
        .in-macos #nav-panel {
          margin-top: 0;
          height: calc(100vh - var(--nav-height));
        }
        .in-macos #nav-header {
          display: none;
        }
        .in-macos #set-header,
        .in-macos #chat-header{
            margin-left: 1rem;
        }
        .in-macos .history-header{
            text-indent: 4rem;
        }
        @media (max-width: 960px) {
            .in-macos #set-header,
            .in-macos #chat-header,
            .in-macos #main-header {
                display: none;
            }
        }
        /*.in-tauri #menu-panel {
            height: 100%;
            padding-top: var(--nav-height);
        }
        .in-tauri #menu-title {
            display: none;
        }
        .in-tauri .mini #menu-title>h1 {
            display: block;
        }
        .in-tauri.hide-menu #set-header,
        .in-tauri.hide-menu #chat-header,
        .in-tauri.hide-menu #main-header {
            padding-left: 72px;
        }
        @media (max-width: 850px) {
            .in-tauri #set-header,
            .in-tauri #chat-header,
            .in-tauri #main-header {
                padding-left: 72px!important;
            }
        }*/
        /* tauri 部分处理 end */
    `;

    console.log('inject js & style')
    window.swiflow = 'swiflow is best'
    document.body.appendChild(style);
    document.body.classList.add('in-tauri');
    if (navigator.userAgent.indexOf('Mac') !== -1) {
        document.body.classList.add('in-macos');
    }
    var decorum = document.querySelector("[data-tauri-decorum-tb]");
    if (decorum && decorum.style?.height) {
      decorum.style.height = "var(--nav-height)";
    }
});