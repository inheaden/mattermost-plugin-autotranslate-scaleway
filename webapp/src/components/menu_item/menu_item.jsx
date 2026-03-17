import React from 'react';
import PropTypes from 'prop-types';

const MenuItem = ({activated, label = 'Translate'}) => {
    if (!activated) {
        return null;
    }

    return (
        <button
            className='style--none'
            role='presentation'
        >
            <span className='MenuItem__icon'>
                <i className='icon fa fa-language'/>
            </span>
            <span>{label}</span>
        </button>
    );
};

MenuItem.propTypes = {
    activated: PropTypes.bool,
    label: PropTypes.string,
};

export default MenuItem;
