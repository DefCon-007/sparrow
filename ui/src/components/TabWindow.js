import React, { PropTypes } from 'react';
import { connect } from 'react-redux';
import { Tab, Tabs, TabList, TabPanel } from 'react-tabs';
import ChatWindow from './ChatWindow.js';
import SearchWindow from './SearchWindow.js';
import { selectTab } from '../actions';

const TabWindowComp = ({tabs, handleSelect, selectedIndex}) => (
    <Tabs onSelect={handleSelect} selectedIndex={selectedIndex}>
      <TabList>
        {tabs.map((tab) =>
          <Tab key={tab.key}>{tab.name}</Tab>
        )}
      </TabList>
      {tabs.map((tab) =>
        <TabPanel key={tab.key}>
          {tab.comp}
        </TabPanel>
      )}
    </Tabs>
);

TabWindowComp.propTypes = {
    tabs: PropTypes.arrayOf(PropTypes.shape({
        name: PropTypes.string.isRequired,
        key: PropTypes.oneOfType([
            PropTypes.string,
            PropTypes.object
        ])
    })).isRequired,
    handleSelect: PropTypes.func.isRequired,
    selectedIndex: PropTypes.number.isRequired
};

const tabsFromState = (state) => {
    let stateTabs = state.tabs.toJS();
    let messages = state.messages.toJS();
    let tabs = [];
    for (var i = 0; i < stateTabs.tabList.length; i++) {
        const tab = stateTabs.tabList[i];
        if (tab.type === 'hubMessages') {
            tabs.push({
                name: tab.name,
                comp: <ChatWindow chatMessages={messages.hubMessages} />,
                key: tab.key
            });
        } else if (tab.type === 'privateMessages') {
            tabs.push({
                name: tab.name,
                comp: <ChatWindow chatMessages={messages.privateMessages[tab.key] || []} />,
                key: tab.key
            });
        } else if (tab.type === 'search') {
            tabs.push({
                name: tab.name,
                comp: <SearchWindow searchText={tab.key}/>,
                key: tab.key
            });
        }
    }
    return tabs;
};

const handleSelect = (dispatch) => {
    return (index) => {
        dispatch(selectTab(index));
    };
};

const selectedIndex = (state) => {
    let stateTabs = state.tabs.toJS();
    let focused = stateTabs.focused;
    if (!focused) {
        return 0;
    }
    for (let i = 0; i < stateTabs.tabList.length; ++i) {
        let tab = stateTabs.tabList[i];
        if (tab.type === focused.type && tab.key === focused.key) {
            return i;
        }
    }
    return 0;
};

const mapStateToProps = (state) => {
    return {
        tabs: tabsFromState(state),
        selectedIndex: selectedIndex(state)
    };
};

const mapDispatchToProps = (dispatch) => {
    return {
        handleSelect: handleSelect(dispatch)
    };
};

const TabWindow = connect(
    mapStateToProps,
    mapDispatchToProps
)(TabWindowComp);

export default TabWindow;
