import ErrorBoundary from './error_boundary';
import MenuItem from './menu_item';

const TranslateThreadMenuItem = () => {
    return (
        <ErrorBoundary>
            <MenuItem label='Translate Thread'/>
        </ErrorBoundary>
    );
};

export default TranslateThreadMenuItem;
