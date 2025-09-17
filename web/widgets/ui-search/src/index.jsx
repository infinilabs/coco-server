import Fullscreen from './Fullscreen';  
import { createCache, StyleProvider } from '@ant-design/cssinjs';
import React from 'react';

export default (props) => {
    const { shadow } = props;

    return (
        <React.StrictMode>
            <StyleProvider container={shadow} cache={createCache()}>
                <Fullscreen {...props} root={shadow || document}/>
            </StyleProvider>
        </React.StrictMode>
    );
}