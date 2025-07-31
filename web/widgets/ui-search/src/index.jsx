import { createCache, StyleProvider } from '@ant-design/cssinjs';
import React from 'react';
import './index.css';
import { default as Page } from './FullscreenPage'
import { default as Modal } from './FullscreenModal'

export const FullscreenPage = (props) => {
    const { shadow } = props;

    return (
        <React.StrictMode>
            <StyleProvider container={shadow} cache={createCache()}>
                <Page {...props} root={shadow || document}/>
            </StyleProvider>
        </React.StrictMode>
    );
}

export const FullscreenModal = (props) => {
    const { shadow } = props;

    return (
        <React.StrictMode>
            <StyleProvider container={shadow} cache={createCache()}>
                <Modal {...props} root={shadow || document}/>
            </StyleProvider>
        </React.StrictMode>
    );
}