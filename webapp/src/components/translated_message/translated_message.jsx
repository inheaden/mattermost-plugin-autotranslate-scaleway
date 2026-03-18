import React from 'react';
import PropTypes from 'prop-types';

export default class TranslatedMessage extends React.PureComponent {
    containerRef = React.createRef();

    originalMessageHtml = null;

    replacedMessageElement = null;

    static propTypes = {
        activated: PropTypes.bool.isRequired,
        translation: PropTypes.object,
        hideTranslatedMessage: PropTypes.func.isRequired,
        onHeightChange: PropTypes.func,
    }

    static defaultProps = {
        activated: false,
    }

    componentDidMount() {
        this.syncTranslatedPostContent();
    }

    componentDidUpdate(prevProps) {
        const nextTranslation = this.props.translation;
        const prevTranslation = prevProps.translation;

        if (nextTranslation &&
            prevTranslation &&
            nextTranslation.translated_text !== prevTranslation.translated_text
        ) {
            this.props.onHeightChange(1);
        }

        this.syncTranslatedPostContent();
    }

    componentWillUnmount() {
        this.restoreOriginalPostContent();
    }

    handleCloseMessage = () => {
        this.restoreOriginalPostContent();
        this.props.hideTranslatedMessage(this.props.translation.post_id);
        this.props.onHeightChange(1);
    }

    getPostMessageElement() {
        const container = this.containerRef.current;

        if (!container) {
            return null;
        }

        const postRoot = container.closest('[data-postid]') || container.closest('.post');
        if (!postRoot) {
            return null;
        }

        return postRoot.querySelector('.post-message__text');
    }

    restoreOriginalPostContent() {
        if (!this.replacedMessageElement || this.originalMessageHtml === null) {
            return;
        }

        this.replacedMessageElement.innerHTML = this.originalMessageHtml;
        this.replacedMessageElement = null;
        this.originalMessageHtml = null;
    }

    syncTranslatedPostContent() {
        const {translation, activated} = this.props;

        if (!activated || !translation || !translation.show || translation.errorMessage) {
            this.restoreOriginalPostContent();
            return;
        }

        const messageElement = this.getPostMessageElement();
        if (!messageElement) {
            return;
        }

        if (this.replacedMessageElement !== messageElement) {
            this.restoreOriginalPostContent();
            this.originalMessageHtml = messageElement.innerHTML;
            this.replacedMessageElement = messageElement;
        }

        const wrapper = document.createElement('div');
        const header = document.createElement('p');
        const icon = document.createElement('i');
        const label = document.createElement('span');
        const link = document.createElement('a');
        const body = document.createElement('div');

        icon.className = 'icon fa fa-language';
        label.textContent = '  Translated  ';
        link.textContent = 'See original';
        link.href = '#';
        link.onclick = (event) => {
            event.preventDefault();
            this.handleCloseMessage();
        };
        body.textContent = translation.translated_text;

        header.appendChild(icon);
        header.appendChild(label);
        header.appendChild(link);
        wrapper.appendChild(header);
        wrapper.appendChild(body);

        while (messageElement.firstChild) {
            messageElement.removeChild(messageElement.firstChild);
        }
        messageElement.appendChild(wrapper);
    }

    renderMessage(message, linkText = '(close)') {
        return (
            <React.Fragment>
                <p>
                    <i className='icon fa fa-language'/>
                    {message}
                    <a onClick={this.handleCloseMessage}>{linkText}</a>
                </p>
            </React.Fragment>
        );
    }

    render() {
        const {translation, activated} = this.props;

        if (!activated || !translation || !translation.show) {
            return null;
        }

        if (translation.errorMessage) {
            return this.renderMessage(
                <span style={{color: 'red'}}>{`  ${translation.errorMessage}  `}</span>,
            );
        }

        return <span ref={this.containerRef}/>;
    }
}
