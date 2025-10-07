import { StyleProvider } from "@ant-design/cssinjs";
import React from "react";

import { default as Page } from "./FullscreenPage";
import { default as Modal } from "./FullscreenModal";

import "./index.css";

export const FullscreenPage = (props) => {
  const { shadow } = props;

  return (
    <React.StrictMode>
      <StyleProvider container={shadow}>
        <Page {...props} root={shadow || document} />
      </StyleProvider>
    </React.StrictMode>
  );
};

export const FullscreenModal = (props) => {
  const { shadow } = props;

  return (
    <React.StrictMode>
      <StyleProvider container={shadow}>
        <Modal {...props} root={shadow || document} />
      </StyleProvider>
    </React.StrictMode>
  );
};
