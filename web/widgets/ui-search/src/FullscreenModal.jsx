import { cloneElement, useState } from 'react';
import FullscreenPage from './FullscreenPage';
import { Modal } from 'antd';
import SearchButton from './SearchButton';

const FullscreenModal = (props) => {

    const { children, ...rest } = props;

    const { root } = rest

    const [visible, setVisible] = useState(false)

    return (
        <>
            {
                children ? cloneElement(children, {
                    onClick: () => setVisible(true)
                }) : (
                    <div style={{ minWidth: 42, maxWidth: '100%' }} onClick={() => setVisible(true)}>
                        <SearchButton placeholder={props.placeholder} />
                    </div>
                )
            }
            <Modal 
                open={visible} 
                style={{ top: 0, margin: 0, padding: 0, maxWidth: '100vw' }}
                width={'100%'}
                onCancel={() => setVisible(false)}
                destroyOnClose
                getContainer={root}
                footer={null}
                styles={{ 
                    content: { padding: 0, minHeight: '100vh' }
                }}
                keyboard={true}
            >
                <FullscreenPage {...rest}/>
            </Modal>
        </>
    );
};

export default FullscreenModal;