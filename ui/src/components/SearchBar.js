import React, { PropTypes } from 'react';
import { connect } from 'react-redux';
import Radium from 'radium';
import { reduxForm } from 'redux-form';
import { fetchSearchResults, newTabMaybe, focusTab } from '../actions';

const submit = ({searchText}, dispatch) => {
    dispatch(fetchSearchResults(searchText));
    dispatch(newTabMaybe(searchText, 'search', searchText));
    dispatch(focusTab('search', searchText));
};

const colors = {
    ss: '#99B898',
    cs: '#FECEA8',
    cdg: '#2A363B',
    cc: '#FF847C',
    mj: '#E84A5F'
};

const styles = {
    base: {
        marginLeft: '350px',
        top: 0,
        right: 0,
        left: 0,
        position: 'fixed',
        backgroundColor: '#FFF',
        height: '100px',
        borderBottom: '1px #DDD solid',
        textAlign: 'center'
    },
    searchBox: {
        position: 'absolute',
        right: '30px',
        margin: '20px 10px 10px 10px',
        fontSize: '2em',
        padding: '10px',
        borderRadius: '10px',
        border: '1px solid #ccc',
        outline: 'none',
        fontWeight: 300,
        ':focus': {
            border: '1px solid ' + colors.cs,
            borderRadius: '10px',
            outline: 'none',
            boxShadow: '0px 0px 2px 0px ' + colors.cs,
            backgroundColor: '#FFF'
        },
        width: '40%',
        backgroundColor: '#FFF'
    },
    title: {
        display: 'inline-block',
        float: 'left',
        position: 'relative',
        top: '50%',
        transform: 'translateY(-50%)',
        marginLeft: '5%',
        fontSize: '2em',
        fontWeight: 300,
        maxWidth: '40%',
        textAlign: 'left'
    }
};

const styleTitle = (title) => {
    if (title.length < 80) {
        return styles.title;
    } else {
        // Haxx
        return {...styles.title, fontSize: `${2 / Math.log(title.length / 20)}em`};
    }
};

const SearchBarComp = ({fields: {searchText}, handleSubmit, title}) => (
    <div style={styles.base}>
      <div style={styleTitle(title)}>{title}</div>
      <form onSubmit={handleSubmit(submit)}>
        <input
           type="text"
           placeholder="Search"
           {...searchText}
           style={styles.searchBox}
           autoComplete="off"
        />
      </form>
    </div>
);

SearchBarComp.propTypes = {
    fields: PropTypes.shape({
        searchText: PropTypes.object.isRequired
    }).isRequired,
    handleSubmit: PropTypes.func.isRequired
};

const getTitleFromState = (state) => {
    const focused = state.tabs.get('focused');
    if (!focused) {
        return "";
    } else if (focused.get('type') === 'search') {
        return `Results for "${focused.get('name')}"`;
    } else {
        return focused.get('name');
    }
};

const mapStateToProps = (state) => {
    return {
        title: getTitleFromState(state)
    };
};

const SearchBar = connect(
    mapStateToProps
)(reduxForm({
    form: 'searchBar',
    fields: ['searchText']
})(Radium(SearchBarComp)));

export default SearchBar;
